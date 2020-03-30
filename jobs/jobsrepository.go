package jobs

import "context"

type Repository interface {
	Create(ctx context.Context, account, group string, job *Job) (*Job, error)
	Delete(ctx context.Context, account, group, id string) error
	Get(ctx context.Context, account, group, id string) (*Job, error)
	List(ctx context.Context, account, group string) ([]string, error)
	Update(ctx context.Context, account, group, id string, job *Job) (*Job, error)
}
