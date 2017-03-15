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

func (h *handler) EnqueueDownlink(appDownlink *types.DownlinkMessage) (err error) {
	appID, devID := appDownlink.AppID, appDownlink.DevID

	ctx := h.Ctx.WithFields(log.Fields{
		"AppID": appID,
		"DevID": devID,
	})
	start := time.Now()
	defer func() {
		if err != nil {
			ctx.WithError(err).Warn("Could not enqueue downlink")
		} else {
			ctx.WithField("Duration", time.Now().Sub(start)).Debug("Enqueued downlink")
		}
	}()

	dev, err := h.devices.Get(appID, devID)
	if err != nil {
		return err
	}

	// Clear redundant fields
	appDownlink.AppID = ""
	appDownlink.DevID = ""

	dev.StartUpdate()
	dev.NextDownlink = appDownlink
	err = h.devices.Set(dev)
	if err != nil {
		return err
	}

	h.mqttEvent <- &types.DeviceEvent{
		AppID: appID,
		DevID: devID,
		Event: types.DownlinkScheduledEvent,
	}

	return nil
}

func (h *handler) HandleDownlink(appDownlink *types.DownlinkMessage, downlink *pb_broker.DownlinkMessage) error {
	appID, devID := appDownlink.AppID, appDownlink.DevID

	ctx := h.Ctx.WithFields(log.Fields{
		"AppID":  appID,
		"DevID":  devID,
		"AppEUI": downlink.AppEui,
		"DevEUI": downlink.DevEui,
	})

	var err error
	defer func() {
		if err != nil {
			h.mqttEvent <- &types.DeviceEvent{
				AppID: appID,
				DevID: devID,
				Event: types.DownlinkErrorEvent,
				Data:  types.ErrorEventData{Error: err.Error()},
			}
			ctx.WithError(err).Warn("Could not handle downlink")
		}
	}()

	dev, err := h.devices.Get(appID, devID)
	if err != nil {
		return err
	}

	// Get Processors
	processors := []DownlinkProcessor{
		h.ConvertFieldsDown,
		h.ConvertToLoRaWAN,
	}

	ctx.WithField("NumProcessors", len(processors)).Debug("Running Downlink Processors")
	downlink.Trace = downlink.Trace.WithEvent("process downlink")

	// Run Processors
	for _, processor := range processors {
		err = processor(ctx, appDownlink, downlink, dev)
		if err == ErrNotNeeded {
			err = nil
			return nil
		} else if err != nil {
			return err
		}
	}

	h.status.downlink.Mark(1)

	ctx.Debug("Send Downlink")

	downlink.Trace = downlink.Trace.WithEvent(trace.ForwardEvent, "broker", h.ttnBrokerID)

	h.downlink <- downlink

	downlinkConfig := types.DownlinkEventConfigInfo{}

	if downlink.DownlinkOption.ProtocolConfig != nil {
		if lorawan := downlink.DownlinkOption.ProtocolConfig.GetLorawan(); lorawan != nil {
			downlinkConfig.Modulation = lorawan.Modulation.String()
			downlinkConfig.DataRate = lorawan.DataRate
			downlinkConfig.BitRate = uint(lorawan.BitRate)
			downlinkConfig.FCnt = uint(lorawan.FCnt)
		}
	}
	if gateway := downlink.DownlinkOption.GatewayConfig; gateway != nil {
		downlinkConfig.Frequency = uint(downlink.DownlinkOption.GatewayConfig.Frequency)
		downlinkConfig.Power = int(downlink.DownlinkOption.GatewayConfig.Power)
	}

	h.mqttEvent <- &types.DeviceEvent{
		AppID: appDownlink.AppID,
		DevID: appDownlink.DevID,
		Event: types.DownlinkSentEvent,
		Data: types.DownlinkEventData{
			Payload:   downlink.Payload,
			GatewayID: downlink.DownlinkOption.GatewayId,
			Config:    downlinkConfig,
		},
	}

	return nil
}
