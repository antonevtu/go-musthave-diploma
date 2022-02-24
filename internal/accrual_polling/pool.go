package accrual_polling

import (
	"context"
	"errors"
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"golang.org/x/sync/errgroup"
	"log"
	"time"
)

type PollT struct {
	ProdChan chan string
	g        *errgroup.Group
	ctx      context.Context
	db       interface{}
	ErrCh    chan error
}

type Poller interface {
	OldestFromQueue(ctx context.Context) (order string, err error)
}

func New(ctx context.Context, repo Poller) PollT {
	prodChan := make(chan string, 1)
	g, ctx := errgroup.WithContext(ctx)
	errCh := make(chan error)
	poll := PollT{
		ProdChan: prodChan,
		g:        g,
		ctx:      ctx,
		ErrCh:    errCh,
	}
	go poll.RunWorkers(repo)
	go poll.RunProd(repo)
	return poll
}

func (p PollT) RunWorkers(repo Poller) {
	numWorkers := 4

	for i := 0; i < numWorkers; i++ {
		p.g.Go(func() error {
			for {
				select {
				case order := <-p.ProdChan:
					err := processOrder(p.ctx, order)
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

func (p PollT) RunProd(repo Poller) {
	p.g.Go(func() error {
		for {
			select {
			case <-p.ctx.Done():
				return nil
			default:
				order, err := repo.OldestFromQueue(p.ctx)
				if errors.Is(err, repository.ErrEmptyQueue) {
					time.Sleep(3 * time.Second)
				} else if err != nil {
					return err
				}
				p.ProdChan <- order
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

func processOrder(ctx context.Context, order string) error {

}
