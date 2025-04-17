package main

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/asticode/go-astits"
)

type MpegTSWriter struct {
	muxer    *astits.Muxer
	videoPID uint16
	startDTS time.Time
}

func NewMpegTSWriter(w io.Writer) (*MpegTSWriter, error) {
	const videoPID = 256

	muxer := astits.NewMuxer(context.Background(), w)
	err := muxer.AddElementaryStream(astits.PMTElementaryStream{
		ElementaryPID: videoPID,
		StreamType:    astits.StreamTypeH264Video,
	})
	if err != nil {
		return nil, err
	}

	muxer.SetPCRPID(videoPID)

	return &MpegTSWriter{
		muxer:    muxer,
		videoPID: videoPID,
		startDTS: time.Now(),
	}, nil
}

func (m *MpegTSWriter) WriteNAL(nalu []byte, pts time.Duration) error {
	buf := &bytes.Buffer{}
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})
	buf.Write(nalu)

	elapsed := time.Since(m.startDTS)
	// 90kHz
	pts90kHz := int64(elapsed / time.Millisecond * 90)

	pes := &astits.PESData{
		Header: &astits.PESHeader{
			OptionalHeader: &astits.PESOptionalHeader{
				PTS: &astits.ClockReference{Base: pts90kHz},
				DTS: &astits.ClockReference{Base: pts90kHz},
			},
		},
		Data: buf.Bytes(),
	}

	_, err := m.muxer.WriteData(&astits.MuxerData{
		PID: m.videoPID,
		PES: pes,
	})
	return err
}

func (m *MpegTSWriter) Close() error {
	return nil
}
