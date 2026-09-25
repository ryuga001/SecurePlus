package provider

import (
	"context"
	"log/slog"
)

const MaxFolderDepth = 64

type folderRef struct {
	id   string
	path string
}

type folderPage struct {
	files   []File
	folders []folderRef
	next    string
}

type pageFetcher func(ctx context.Context, folder folderRef, cursor string) (folderPage, error)

type walkFrame struct {
	folder  folderRef
	cursor  string
	started bool
	pending []folderRef
}

func walk(ctx context.Context, root folderRef, fetch pageFetcher, emit func(File) error) error {
	stack := []*walkFrame{{folder: root}}
	skipped := 0

	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		top := stack[len(stack)-1]

		if count := len(top.pending); count > 0 {
			child := top.pending[count-1]
			top.pending = top.pending[:count-1]

			if len(stack) >= MaxFolderDepth {
				skipped++

				continue
			}

			stack = append(stack, &walkFrame{folder: child})

			continue
		}

		if top.started && top.cursor == "" {
			stack = stack[:len(stack)-1]

			continue
		}

		page, err := fetch(ctx, top.folder, top.cursor)
		if err != nil {
			return err
		}

		top.started = true
		top.cursor = page.next
		top.pending = page.folders

		for _, file := range page.files {
			if err := emit(file); err != nil {
				return err
			}
		}
	}

	if skipped > 0 {
		slog.WarnContext(ctx, "folders deeper than the scanner limit were skipped",
			"max_depth", MaxFolderDepth,
			"skipped_folders", skipped,
		)
	}

	return nil
}

func childPath(parent, name string) string {
	if parent == "" {
		return name
	}

	return parent + "/" + name
}
