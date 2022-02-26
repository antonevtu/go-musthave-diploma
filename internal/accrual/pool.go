package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net/http"
	"time"
)

type PollT struct {
	ProdChan    chan string
	g           *errgroup.Group
	ctx         context.Context
	db          interface{}
	ErrCh       chan error
	serviceAddr string
}

type Poller interface {
	OldestFromQueue(ctx context.Context) (order string, err error)
	DeferOrder(ctx context.Context, order, status string) error
	FinalizeOrder(ctx context.Context, order, status string, accrual float64) error
}

func New(ctx context.Context, repo Poller, cfgApp cfg.Config) PollT {
	prodChan := make(chan string, 1)
	g, ctx := errgroup.WithContext(ctx)
	errCh := make(chan error)
	poll := PollT{
		ProdChan:    prodChan,
		g:           g,
		ctx:         ctx,
		ErrCh:       errCh,
		serviceAddr: cfgApp.AccrualSystemAddress + "/api/orders/",
	}
	go poll.RunWorkers(repo)
	go poll.RunProducer(repo)
	return poll
}

func (p PollT) RunWorkers(repo Poller) {
	numWorkers := 4

	for i := 0; i < numWorkers; i++ {
		p.g.Go(func() error {
			for {
				select {
				case order := <-p.ProdChan:
					err := p.processOrderAccrual(repo, order)
					if err != nil {
						return err
					}
				case <-p.ctx.Done():
					return nil
				}
			}
		})
	}

	if err := p.g.Wait(); err != nil {
		p.ErrCh <- fmt.Errorf("error in worker pool: %w", err)
	}
}

func (p PollT) RunProducer(repo Poller) {
	p.g.Go(func() error {
		for {
			select {
			case <-p.ctx.Done():
				return nil
			default:
				order, err := repo.OldestFromQueue(p.ctx)
				if err == nil {
					p.ProdChan <- order
				} else if errors.Is(err, repository.ErrEmptyQueue) {
					fmt.Println("Ожидание 1с")
					time.Sleep(1 * time.Second)
				} else {
					return err
				}
			}
		}
	})

	if err := p.g.Wait(); err != nil {
		p.ErrCh <- fmt.Errorf("error in producer goroutine: %w", err)
	}
}

func (p PollT) Close() {
	_ = p.g.Wait()
	log.Println("accrual pool has closed")
}

type serviceResponce struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func (p PollT) processOrderAccrual(repo Poller, order string) error {

	// make request to service
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, p.serviceAddr+order, bytes.NewBufferString(""))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		res := serviceResponce{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			return err
		}

		err = repo.FinalizeOrder(p.ctx, order, res.Status, res.Accrual)
		if err != nil {
			return err
		}

	case http.StatusTooManyRequests:
		err := repo.DeferOrder(p.ctx, order, "")
		if err != nil {
			return err
		}
		time.Sleep(60 * time.Second)

	default:
		err := repo.DeferOrder(p.ctx, order, "")
		if err != nil {
			return err
		}
	}

	return nil
}
