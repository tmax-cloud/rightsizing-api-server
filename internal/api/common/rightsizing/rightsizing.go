package rightsizing

import (
	"context"

	"rightsizing-api-server/internal/api/common/resource"
	grpcclient "rightsizing-api-server/internal/grpc"
)

func Rightsizing(ctx context.Context, client *grpcclient.Client, info *resource.ResourceUsageInfo) error {
	resp, err := client.Rightsizing(ctx, info.Usage)
	if err != nil {
		return err
	}
	info.OptimizedUsage = resp.Result
	return nil
}
