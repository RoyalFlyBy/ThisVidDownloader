package thisvid_downloader

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gokhttp_download "github.com/BRUHItsABunny/gOkHttp-download"
	gokhttp_requests "github.com/BRUHItsABunny/gOkHttp/requests"
	"github.com/RoyalFlyBy/ThisVidDownloader/utils"
	thisvidapi "github.com/RoyalFlyBy/go-thisvid/api"
	thisvidc "github.com/RoyalFlyBy/go-thisvid/client"
	"github.com/RoyalFlyBy/go-thisvid/constants"
)

// App wires the command-line args to the ThisVid client and the download tasks.
type App struct {
	Args   *utils.AppArgs
	Client *thisvidc.ThisVidClient
	HTTP   *http.Client
	Global *gokhttp_download.GlobalDownloadTracker
	Log    *slog.Logger
}

// Run dispatches to the first target that was set and renders progress until
// every download finishes.
func (app *App) Run(ctx context.Context) error {
	if err := os.MkdirAll(app.Args.GetOutput(), 0755); err != nil { // the task creates dirs with 0600
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	uiCtx, stopUI := context.WithCancel(ctx)
	go app.RenderUI(uiCtx)
	defer stopUI()

	var err error
	switch {
	case app.Args.DoVideo():
		err = app.DownloadVideo(ctx)
	case app.Args.DoList():
		err = app.DownloadList(ctx)
	case app.Args.DoMember():
		err = app.DownloadMember(ctx)
	case app.Args.DoPlaylist():
		err = app.DownloadPlaylist(ctx)
	default:
		err = fmt.Errorf("nothing to do, see --help")
	}

	app.Global.Stop() // ends the tracker goroutines
	return err
}

// DownloadVideo downloads one video given a page URL, path or slug.
func (app *App) DownloadVideo(ctx context.Context) error {
	app.Log.Info("Trigger: download video", "video", app.Args.GetVideo())
	return app.processVideo(ctx, app.Args.GetVideo())
}

// DownloadList downloads every video listed in a text file, one per line.
// Blank lines and lines starting with '#' are ignored.
func (app *App) DownloadList(ctx context.Context) error {
	app.Log.Info("Trigger: download list", "list", app.Args.GetList())
	f, err := os.Open(app.Args.GetList())
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		if err = app.processVideo(ctx, line); err != nil {
			app.Log.Error("app.processVideo", "video", line, "err", err) // one bad item doesn't stop the list
		}
	}
	return scanner.Err()
}

// DownloadMember downloads from a member's public videos, newest first.
func (app *App) DownloadMember(ctx context.Context) error {
	app.Log.Info("Trigger: download member", "member", app.Args.GetMember())
	member := app.Args.GetMember()
	return app.downloadCollection(ctx, func(page int) (*thisvidapi.VideoList, error) {
		return app.Client.MemberVideos(ctx, member, page)
	})
}

// DownloadPlaylist downloads from a playlist.
func (app *App) DownloadPlaylist(ctx context.Context) error {
	app.Log.Info("Trigger: download playlist", "playlist", app.Args.GetPlaylist())
	playlist := app.Args.GetPlaylist()
	return app.downloadCollection(ctx, func(page int) (*thisvidapi.VideoList, error) {
		return app.Client.PlaylistVideos(ctx, playlist, page)
	})
}

type listFetcher func(page int) (*thisvidapi.VideoList, error)

// downloadCollection pages through a listing, skipping offset items and stopping
// after amount downloads or at the last page.
func (app *App) downloadCollection(ctx context.Context, fetch listFetcher) error {
	amount, offset := app.Args.GetAmount(), app.Args.GetOffset()
	seen, done, page := 0, 0, 1
	for done < amount {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		list, err := fetch(page)
		if err != nil {
			return fmt.Errorf("fetch page %d: %w", page, err)
		}
		if len(list.Items) == 0 {
			break // an empty page is the end, not an error
		}

		for _, item := range list.Items {
			seen++
			if seen <= offset {
				continue
			}
			if done >= amount {
				break
			}
			if err = app.processVideo(ctx, item.Slug); err != nil {
				app.Log.Error("app.processVideo", "video", item.Slug, "err", err)
			}
			done++
		}

		if list.NextPage == 0 {
			break
		}
		page = list.NextPage
	}
	return nil
}

// processVideo resolves one video, picks a stream and downloads it unless the
// final file already exists.
func (app *App) processVideo(ctx context.Context, identifier string) error {
	video, err := app.Client.GetVideo(ctx, identifier)
	if err != nil {
		return fmt.Errorf("app.Client.GetVideo: %w", err)
	}
	if len(video.Streams) == 0 {
		if video.Private {
			return fmt.Errorf("video %q is private; pass --cookies from a logged-in friend's session", video.Slug)
		}
		return fmt.Errorf("video %q has no streams", video.Slug)
	}

	stream := utils.SelectStream(video.Streams, app.Args.GetQuality())
	if stream == nil {
		return fmt.Errorf("video %q has no usable stream", video.Slug)
	}

	direct, err := app.Client.GetDirectURL(ctx, stream.URL)
	if err != nil {
		return fmt.Errorf("app.Client.GetDirectURL: %w", err)
	}
	if direct.Size <= 0 {
		return fmt.Errorf("could not determine the size of %q", stream.URL)
	}

	extension := utils.StreamExtension(stream, direct.ContentType)
	fileLocation := app.outputPath(video, extension)

	// The threaded task writes ranged chunks to .partN and creates the target
	// empty until it merges them, while a single-thread run resumes the target
	// itself. Only the expected size proves a file is complete.
	if size, ok := utils.FileSize(fileLocation); ok {
		switch {
		case size == direct.Size:
			app.Log.Info("Skipping, already complete", "file", fileLocation)
			return nil
		case size > direct.Size:
			app.Log.Info("Truncating oversized target", "file", fileLocation, "size", size)
			if err = os.Remove(fileLocation); err != nil {
				return fmt.Errorf("os.Remove: %w", err)
			}
		}
	}

	if err = utils.EnsureDir(fileLocation); err != nil {
		return fmt.Errorf("utils.EnsureDir: %w", err)
	}
	return app.downloadStream(ctx, stream, direct, fileLocation)
}

func (app *App) outputPath(video *thisvidapi.Video, extension string) string {
	name := utils.VideoBaseName(video, app.Args.GetPrefixID())
	return filepath.Join(app.Args.GetOutput(), name+extension)
}

func (app *App) downloadStream(ctx context.Context, stream *thisvidapi.VideoStream, direct *thisvidapi.DirectURL, fileLocation string) error {
	referer := strings.TrimSpace(direct.Referer)
	if len(referer) == 0 {
		referer = constants.BaseURL + "/"
	}

	task, err := gokhttp_download.NewThreadedDownloadTask(ctx, app.HTTP, app.Global, fileLocation, stream.URL,
		uint64(app.Args.GetThreads()), uint64(direct.Size),
		gokhttp_requests.NewHeaderOption(thisvidapi.StreamHeaders(app.Client.UserAgent, referer)),
	)
	if err != nil {
		return fmt.Errorf("gokhttp_download.NewThreadedDownloadTask: %w", err)
	}
	if task == nil {
		return fmt.Errorf("gokhttp_download.NewThreadedDownloadTask returned a nil task")
	}

	app.Log.Info("Downloading", "file", fileLocation, "size", direct.Size, "stream", stream.Label)
	if err = task.Download(ctx); err != nil {
		return fmt.Errorf("task.Download: %w", err)
	}
	return nil
}

// RenderUI clears the terminal once a second and prints the shared tracker.
func (app *App) RenderUI(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Print("\033[H\033[2J" + app.Global.Tick(true))
		}
	}
}
