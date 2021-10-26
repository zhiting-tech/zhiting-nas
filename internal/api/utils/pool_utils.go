package utils

import (
	"context"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"google.golang.org/grpc"
)

// PartitionInfo 存储池分区详情
type PartitionInfo struct {
	PoolName   string
	Name 	   string
	Size 	   int64
	FreeSize   int64
	VgFreeSize int64
}

// GetPartitionInfo 获取分区详情
func GetPartitionInfo(poolName, partitionName string) (*PartitionInfo, error) {
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()
	groups, err := client.VolumeGroupList(ctx, &proto.Empty{})
	if err != nil {
		return nil, err
	}
	for _, vg := range groups.VGS {
		if vg.Name != poolName {
			// 使用名称进行匹配
			continue
		}
		for _, lv := range vg.LVS {
			if lv.Name == partitionName {
				partitionInfo := PartitionInfo{
					PoolName: poolName,
					Name: lv.Name,
					Size: lv.Size,
					FreeSize: lv.FreeSize,
					VgFreeSize: vg.FreeSize,
				}
				return &partitionInfo, nil
			}
		}
	}

	return nil, errors.New(status.PartitionInfoErr)
}