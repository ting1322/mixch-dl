package inter

import "context"

type Live interface {
	WaitStreamStart(ctx context.Context, conn INet) error
	Download(ctx context.Context, netconn INet, fio IFs, filename string) error
}
