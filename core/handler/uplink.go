// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package handler

import (
	"time"

	pb_broker "github.com/TheThingsNetwork/ttn/api/broker"
	"github.com/TheThingsNetwork/ttn/api/trace"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/apex/log"
)

// ResponseDeadline indicates how long
var ResponseDeadline = 100 * time.Millisecond

func (h *handler) HandleUplink(uplink *pb_broker.DeduplicatedUplinkMessage) (err error) {
	appID, devID := uplink.AppId, uplink.DevId
	ctx := h.Ctx.WithFields(log.Fields{
		"AppID": appID,
		"DevID": devID,
	})
	start := time.Now()
	defer func() {
		if err != nil {
			h.mqttEvent <- &types.DeviceEvent{
				AppID: appID,
				DevID: devID,
				Event: types.UplinkErrorEvent,
				Data:  types.ErrorEventData{Error: err.Error()},
			}
			ctx.WithError(err).Warn("Could not handle uplink")
		} else {
			ctx.WithField("Duration", time.Now().Sub(start)).Info("Handled uplink")
		}
	}()
	h.status.uplink.Mark(1)

	uplink.Trace = uplink.Trace.WithEvent(trace.ReceiveEvent)

	dev, err := h.devices.Get(appID, devID)
	if err != nil {
		return err
	}

	// Build AppUplink
	appUplink := &types.UplinkMessage{
		AppID: appID,
		DevID: devID,
	}

	// Get Uplink Processors
	processors := []UplinkProcessor{
		h.ConvertFromLoRaWAN,
		h.ConvertMetadata,
		h.ConvertFieldsUp,
	}

	ctx.WithField("NumProcessors", len(processors)).Debug("Running Uplink Processors")
	uplink.Trace = uplink.Trace.WithEvent("process uplink")

	// Run Uplink Processors
	for _, processor := range processors {
		err = processor(ctx, uplink, appUplink, dev)
		if err == ErrNotNeeded {
			err = nil
			return nil
		} else if err != nil {
			return err
		}
	}

	// Publish Uplink
	h.mqttUp <- appUplink
	if h.amqpEnabled {
		h.amqpUp <- appUplink
	}

	if uplink.ResponseTemplate == nil {
		ctx.Debug("No Downlink Available")
		return nil
	}

	<-time.After(ResponseDeadline)

	// Find scheduled downlink
	var appDownlink types.DownlinkMessage
	dev, err = h.devices.Get(appID, devID)
	if err != nil {
		return err
	}
	if dev.NextDownlink != nil {
		appDownlink = *dev.NextDownlink
	}

	// Prepare Downlink
	downlink := uplink.ResponseTemplate
	downlink.Trace = uplink.Trace.WithEvent("prepare downlink")
	appDownlink.AppID = uplink.AppId
	appDownlink.DevID = uplink.DevId

	// Handle Downlink
	err = h.HandleDownlink(&appDownlink, downlink)
	if err != nil {
		return err
	}

	// Clear Downlink
	dev.StartUpdate()
	dev.NextDownlink = nil
	err = h.devices.Set(dev)
	if err != nil {
		return err
	}

	return nil
}
