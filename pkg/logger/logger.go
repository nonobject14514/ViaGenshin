package logger

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const (
	substr = "ViaGenshin/"
	strlen = len(substr)
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		idx := strings.LastIndex(file, substr)
		if idx != -1 {
			file = file[idx+strlen:]
		}
		return file + ":" + strconv.Itoa(line)
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

var Logger = zerolog.New(zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: time.StampMilli,
}).With().Timestamp().Caller().Logger().Level(zerolog.InfoLevel)

var (
	Trace = Logger.Trace
	Debug = Logger.Debug
	Info  = Logger.Info
	Warn  = Logger.Warn
	Error = Logger.Error
)
