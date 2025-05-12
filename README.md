
[🇰🇷 한국어](#한국어) &nbsp;|&nbsp; [🇺🇸 English](#english)

# English

## Overview

**go-rtsp-tools is a lightweight RTSP utility toolkit written in Go** <br>
It provides real-time stream inspection, health monitoring, TS segment saving, and WebRTC (LiveKit) relay.

# Features

Directory         | Description
------------------|--------------------------------------------------------------------------------------
live-inspector    | Print RTSP SDP and NAL packet info in real time
health-monitor    | Check multiple RTSP streams for RTP packet reception
stream-analyzer   | Analyze stream quality (packet loss, bitrate, IDR frame count, etc.)
livekit-relay     | Relay RTP packets to a LiveKit WebRTC server
ts-dumper         | Save RTSP stream as `.ts` files (with optional segment duration)

# Usage

```shell
# Inspect RTSP SDP and NAL stream
go run ./cmd/live-inspector rtsp://127.0.0.1:554

# Basic health monitor (RTP reception check)
go run ./cmd/health-monitor                            

# Analyze stream quality (loss, bitrate, IDR count, etc.)
go run ./cmd/stream-analyzer                           

# Relay to LiveKit WebRTC (requires RTP on UDP 5004)
go run ./cmd/livekit-relay                             

# Save RTSP stream to TS file (segment duration configurable)
go run ./cmd/ts-dumper rtsp://127.0.0.1:554            
```

# Tech Stack
- Go 1.20+
- RTSP / RTP / SDP / H264
- gortsplib, LiveKit, Viper

# Notes
- Written in pure Go (no FFmpeg or GStreamer)
- Analyzes H264 NAL units directly
- Supports segmenting .ts output using config file
- CLI-based and easy to extend for other use cases

---

## 한국어

## 개요
**Go 기반 RTSP 스트림 분석 및 저장 툴킷** <br>
실시간 RTSP 수신, 분석, TS 저장, Livekit 릴레이 지원가능한 경량 유틸리티

## 구성 기능

| 디렉토리         | 설명                                                                   |
|------------------|------------------------------------------------------------------------|
| `live-inspector` | RTSP SDP + NAL 패킷 실시간 로깅                                        |
| `health-monitor` | 병렬 RTSP 연결 상태 확인 (RTP 수신 여부)                               |
| `stream-analyzer`| RTSP 통계 분석 (손실률, IDR 수, 비트레이트, NAL 분포 등)              |
| `livekit-relay`  | RTSP(RTP) → LiveKit WebRTC 릴레이                                       |
| `ts-dumper`      | RTSP 스트림을 실시간 `.ts` 파일 저장 (초단위 분할 저장 지원)   |


## 설치
```bash
rm go.mod go.sum
go mod init go-rtsp-tools
go mod tidy
```

## 사용예시

```shell
# RTSP SDP + NAL 수신 확인
go run ./cmd/live-inspector rtsp://127.0.0.1:554

# 단순 Health 체크 (RTP 수신 유무 확인)
go run ./cmd/health-monitor

# RTSP 상태 분석 (손실률 / 비트레이트 / IDR 등)
go run ./cmd/stream-analyzer

# LiveKit RTP 릴레이 (UDP 5004번으로 rtsp 전달 필요)
go run ./cmd/livekit-relay

# RTSP → TS 파일 실시간 저장 (단일 저장, 분할 시간 제어 가능)
go run ./cmd/ts-dumper rtsp://127.0.0.1:554
```

## 기술스택
- Go 1.20+
- RTSP / RTP / SDP / H264
- gortsplib / LiveKit
- Viper (설정 관리)

## 프로젝트 특징
- 순수 Go로 구현 – FFmpeg, GStreamer 등 외부 바이너리 없이 동작

- NAL 단위 분석 및 MPEG-TS 직접 생성

- 분할 저장 지원 – config에서 저장 주기(seconds) 설정 가능

- 확장성 – 향후 HLS, mp4 변환, Web UI 연동 등 기능 확장 가능
