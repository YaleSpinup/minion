package jobs

import "context"

type Repository interface {
	Create(ctx context.Context, account string, job *Job) (*Job, error)
	Delete(ctx context.Context, account, id string) error
	Get(ctx context.Context, account, id string) (*Job, error)
	List(ctx context.Context, account string) ([]string, error)
	Update(ctx context.Context, account, id string, job *Job) (*Job, error)
}
