package main

import (
	"fmt"
	"os"
	"time"

	"go-rtsp-tools/internal/config"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/pion/rtp"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ./main.go <rtsp-url>")
		os.Exit(1)
	}

	config.LoadConfig()
	segmentDuration := config.Cfg.TsDump.Duration

	rtspURL := os.Args[1]

	u, err := base.ParseURL(rtspURL)
	if err != nil {
		panic(err)
	}

	client := &gortsplib.Client{}
	err = client.Start(u.Scheme, u.Host)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	desc, _, err := client.Describe(u)
	if err != nil {
		panic(err)
	}

	var h264Format *format.H264
	var selectedMedia *description.Media
	for _, m := range desc.Medias {
		for _, f := range m.Formats {
			if h, ok := f.(*format.H264); ok {
				h264Format = h
				selectedMedia = m
				break
			}
		}
	}
	if h264Format == nil {
		panic("H264 format not found")
	}

	err = client.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		panic(err)
	}
	_, err = client.Play(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("[INFO] Receiving RTP")

	baseTimestamp := time.Now().Format("20060102150405")
	segmentIndex := 1
	segmentDurationPTS := int64(segmentDuration * 90000)
	var basePTS int64 = -1

	openSegment := func(index int) (*os.File, *MpegTSWriter, error) {
		name := baseTimestamp
		if segmentDuration > 0 {
			name = fmt.Sprintf("%s_%d", baseTimestamp, index)
		}
		filename := fmt.Sprintf("%s.ts", name)
		f, err := os.Create(filename)
		if err != nil {
			return nil, nil, err
		}
		w, err := NewMpegTSWriter(f)
		if err != nil {
			return nil, nil, err
		}
		return f, w, nil
	}

	tsFile, writer, err := openSegment(segmentIndex)
	if err != nil {
		panic(err)
	}
	defer tsFile.Close()
	defer writer.Close()

	decoder := &rtph264.Decoder{}
	err = decoder.Init()
	if err != nil {
		panic(err)
	}

	var gotSPS, gotPPS, gotIDR bool
	var spsNALU, ppsNALU []byte
	var wroteHeaders bool

	client.OnPacketRTP(selectedMedia, h264Format, func(pkt *rtp.Packet) {
		nalus, err := decoder.Decode(pkt)
		if err != nil || len(nalus) == 0 {
			return
		}

		pts := int64(pkt.Timestamp)
		if basePTS < 0 {
			basePTS = pts
		}

		if segmentDuration > 0 && pts >= basePTS+(segmentDurationPTS*int64(segmentIndex)) {
			writer.Close()
			tsFile.Close()

			segmentIndex++
			tsFile, writer, err = openSegment(segmentIndex)
			if err != nil {
				panic(err)
			}

			wroteHeaders = false
		}

		for _, nalu := range nalus {
			if len(nalu) < 1 {
				continue
			}

			nalType := nalu[0] & 0x1F
			switch nalType {
			case 7:
				spsNALU = nalu
				gotSPS = true
			case 8:
				ppsNALU = nalu
				gotPPS = true
			case 5:
				gotIDR = true
			}

			if gotSPS && gotPPS && gotIDR {
				if !wroteHeaders && spsNALU != nil && ppsNALU != nil {
					_ = writer.WriteNAL(spsNALU, time.Duration(pts))
					_ = writer.WriteNAL(ppsNALU, time.Duration(pts))
					wroteHeaders = true
				}

				if nalType == 1 || nalType == 5 {
					err := writer.WriteNAL(nalu, time.Duration(pts))
					if err != nil {
						fmt.Printf("[ERROR] writing TS: %v\n", err)
					}
				}
			}
		}
	})

	select {}
}
