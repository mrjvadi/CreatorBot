// Package agentlistener پردازش پیام‌های heartbeat و result از agentmanager.
// هر دو botmanager و apimanager از این پکیج استفاده می‌کنند تا منطق مشترک
// در یک جا باشد و اختلاف پیاده‌سازی بین دو سرویس از بین برود.
package agentlistener

import (
	"context"
	"encoding/json"

	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// HandleHeartbeat پیام heartbeat از agentmanager را پردازش می‌کند:
// - آمار سرور (CPU/RAM/containers JSON) را ذخیره می‌کند
// - وضعیت instance هایی که container آن‌ها در لیست است را به‌روز می‌کند
func HandleHeartbeat(ctx context.Context, data []byte, st *store.Store, log ports.Logger) {
	var hb protocol.HeartbeatMsg
	if err := json.Unmarshal(data, &hb); err != nil {
		return
	}
	containersJSON, _ := json.Marshal(hb.Containers)
	if err := st.RecordHeartbeat(ctx, hb.ServerID, hb.CPUPercent, hb.MemoryUsedMB, hb.MemoryTotalMB, string(containersJSON)); err != nil {
		log.Warn("record heartbeat failed", ports.F("server_id", hb.ServerID), ports.F("err", err))
	}
	for _, c := range hb.Containers {
		switch c.State {
		case "running":
			_ = st.UpdateInstanceStatusByContainerName(ctx, c.Name, models.StatusRunning)
		case "exited", "dead":
			_ = st.UpdateInstanceStatusByContainerName(ctx, c.Name, models.StatusStopped)
		}
	}
}

// HandleResult پیام نتیجه‌ی دستور Docker از agentmanager را پردازش می‌کند:
// - Deploy موفق → StatusRunning
// - هر دستور ناموفق → StatusError
// - Stop → StatusStopped
// - Remove → حذف از DB
func HandleResult(ctx context.Context, data []byte, st *store.Store, log ports.Logger) {
	var result protocol.ResultMsg
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}
	inst, err := st.FindInstanceByContainerName(ctx, result.ContainerName)
	if err != nil || inst == nil {
		return
	}
	switch {
	case result.Success && result.CommandType == string(protocol.MsgDeploy):
		_ = st.UpdateInstanceStatus(ctx, inst.ID, models.StatusRunning)
		log.Info("instance running",
			ports.F("instance", inst.ID),
			ports.F("container", result.ContainerName))
	case !result.Success:
		_ = st.UpdateInstanceStatus(ctx, inst.ID, models.StatusError)
		log.Error("instance failed",
			ports.F("instance", inst.ID),
			ports.F("container", result.ContainerName),
			ports.F("err", result.Error))
	case result.CommandType == string(protocol.MsgStop):
		_ = st.UpdateInstanceStatus(ctx, inst.ID, models.StatusStopped)
	case result.CommandType == string(protocol.MsgRemove):
		_ = st.DeleteInstance(ctx, inst.ID)
	}
}
