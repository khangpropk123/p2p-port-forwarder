package pkg

import "github.com/pion/webrtc/v3"

type DataChanelWriter struct {
	*webrtc.DataChannel
}

func NewDataChanelWriter(rtc *webrtc.DataChannel) *DataChanelWriter {
	return &DataChanelWriter{rtc}
}

func (d *DataChanelWriter) Write(b []byte) (int, error) {
	err := d.DataChannel.Send(b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}
