package main

import (
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	bufSize     = 1024
	numRetries  = 6
	dialTimeOut = 6 * time.Second
)

func handler(conn net.Conn, outConn string) error {
	numRetriesExceeded := errors.New("num retries exceeded")

	for i := 0; i < numRetries; i++ {
		log.Info().Int("attempt", i).Int("timeout", int(dialTimeOut.Seconds())).Msg("tcpforwarder - dial connection")

		target, err := net.DialTimeout("tcp", outConn, dialTimeOut)
		if err != nil {
			log.Warn().Err(err).Msg("tcpforwarder - remote connection")

			continue
		}

		log.Info().Str("outConn", outConn).Msg("tcpforwarder - connection established")

		go func() {
			defer target.Close()

			outBuff := make([]byte, bufSize)

			_, err := io.CopyBuffer(target, conn, outBuff)
			if err != nil {
				// TODO: fix to reuse connections
				log.Error().Err(err).Msg("tcpforwarder - outbuff")
			}

			log.Info().Msg("tcpforwarder - end target connection")
		}()

		go func() {
			defer conn.Close()

			inBuff := make([]byte, bufSize)

			_, err := io.CopyBuffer(conn, target, inBuff)
			if err != nil {
				// TODO: fix to reuse connections
				log.Error().Err(err).Msg("tcpforwarder - inbuff")
			}

			log.Info().Msg("tcpforwarder - end main connection")
		}()

		return nil
	}

	log.Error().Err(numRetriesExceeded).Msg("tcpforwarder - handler")

	return numRetriesExceeded
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var inConn, outConn string

	flag.StringVar(&inConn, "i", "", "incoming connecction -> host:port")
	flag.StringVar(&outConn, "o", "", "output connection forwarded to -> host:port")

	flag.Parse()

	if len(inConn) == 0 || len(outConn) == 0 {
		log.Warn().Str("inConn", inConn).Str("outConn", outConn).Msg("tcpforwarder - missing arg")
		flag.PrintDefaults()
		os.Exit(-1)
	}

	log.Info().Str("inConn", inConn).Str("outConn", outConn).Msg("tcpforwarder - args")

	listener, err := net.Listen("tcp", inConn)
	if err != nil {
		log.Error().Err(err).Msg("tcpforwarder - create listener")

		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("tcpforwarder - accept incomin connection")

			panic(err)
		}

		log.Info().Str("remote addr", conn.RemoteAddr().String()).Msg("tcpforwarder - accepted connection")

		go handler(conn, outConn)
	}
}
