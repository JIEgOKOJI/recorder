package format

import (
	"Recorder/joy4/av/avutil"
	"Recorder/joy4/format/aac"
	"Recorder/joy4/format/flv"
	"Recorder/joy4/format/mp4"
	"Recorder/joy4/format/rtmp"
	"Recorder/joy4/format/rtsp"
	"Recorder/joy4/format/ts"
)

func RegisterAll() {
	avutil.DefaultHandlers.Add(mp4.Handler)
	avutil.DefaultHandlers.Add(ts.Handler)
	avutil.DefaultHandlers.Add(rtmp.Handler)
	avutil.DefaultHandlers.Add(rtsp.Handler)
	avutil.DefaultHandlers.Add(flv.Handler)
	avutil.DefaultHandlers.Add(aac.Handler)
}
