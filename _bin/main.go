package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	gokhttp_download "github.com/BRUHItsABunny/gOkHttp-download"
	gokhttp_client "github.com/BRUHItsABunny/gOkHttp/client"
	downloader "github.com/RoyalFlyBy/ThisVidDownloader"
	"github.com/RoyalFlyBy/ThisVidDownloader/utils"
	thisvid "github.com/RoyalFlyBy/go-thisvid"
)

func main() {
	appArgs := utils.AppArgs{
		Video:    flag.String("video", "", "video page URL, path or slug"),
		List:     flag.String("list", "", "path to a text file with one video URL/slug per line"),
		Member:   flag.String("member", "", "member profile URL or numeric id"),
		Playlist: flag.String("playlist", "", "playlist URL or numeric id"),
		Output:   flag.String("output", ".", "directory to download into"),
		Proxy:    flag.String("proxy", "", "proxy URL, e.g. http://127.0.0.1:8888 or socks5://127.0.0.1:9050"),
		Cookies:  flag.String("cookies", "", `raw Cookie header for private videos, e.g. "PHPSESSID=...; kt_tcookie=1"`),
		Debug:    flag.Bool("debug", false, "write debug logs to thisvid_downloader.log"),
		PrefixID: flag.Bool("pid", false, "prepend [id] to filenames"),
		Threads:  flag.Int("threads", 4, "simultaneous threads per download (1..12)"),
		Amount:   flag.Int("amount", 20, "how many videos to download from a member or playlist"),
		Offset:   flag.Int("offset", 0, "how many videos to skip in a member or playlist (newest first)"),
		Quality:  flag.Int("quality", -1, "stream index, -1 = best available"),
		Version:  flag.Bool("version", false, "print version information and exit"),
	}
	flag.Parse()

	versionOutput := downloader.VersionRoutine()
	if appArgs.GetVersion() {
		fmt.Println(versionOutput)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logFile, err := os.OpenFile("thisvid_downloader.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("os.OpenFile: ", err)
		os.Exit(1)
	}
	defer logFile.Close()
	level := slog.LevelWarn
	if appArgs.GetDebug() {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: level}))
	logger.Debug(versionOutput)

	var opts []gokhttp_client.Option
	if proxy := appArgs.GetProxy(); len(proxy) > 0 {
		opts = append(opts, gokhttp_client.NewProxyOption(proxy))
	}
	hClient, err := gokhttp.NewHTTPClient(opts...)
	if err != nil {
		fmt.Println("gokhttp.NewHTTPClient: ", err)
		os.Exit(1)
	}

	client, err := thisvid.NewThisVidClientWithCookies(hClient, "", appArgs.GetCookies())
	if err != nil {
		fmt.Println("thisvid.NewThisVidClientWithCookies: ", err)
		os.Exit(1)
	}

	app := downloader.App{
		Args:   &appArgs,
		Client: client,
		// The cookie client may be a copy of hClient with a jar attached; tasks
		// must use that copy so friends-only streams get the session cookies.
		HTTP:   client.Client,
		Global: gokhttp_download.NewGlobalDownloadTracker(30 * time.Second),
		Log:    logger,
	}

	if err = app.Run(ctx); err != nil {
		logger.Error("app.Run", "err", err)
		fmt.Println("err: ", err)
		os.Exit(1)
	}
}
