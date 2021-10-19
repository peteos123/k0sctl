package phase

import (
	"fmt"
	"path"

	"github.com/k0sproject/k0sctl/config"
	"github.com/k0sproject/k0sctl/config/cluster"
	"github.com/k0sproject/rig/exec"

	log "github.com/sirupsen/logrus"
)

// UploadFiles implements a phase which upload files to hosts
type UploadFiles struct {
	GenericPhase

	hosts cluster.Hosts
}

// Title for the phase
func (p *UploadFiles) Title() string {
	return "Upload files to hosts"
}

// Prepare the phase
func (p *UploadFiles) Prepare(config *config.Cluster) error {
	p.Config = config
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return len(h.Files) > 0
	})

	return nil
}

// ShouldRun is true when there are workers
func (p *UploadFiles) ShouldRun() bool {
	return len(p.hosts) > 0
}

// Run the phase
func (p *UploadFiles) Run() error {
	return p.Config.Spec.Hosts.ParallelEach(p.uploadFiles)
}

func (p *UploadFiles) uploadFiles(h *cluster.Host) error {
	var resolved []cluster.UploadFile

	for _, f := range h.Files {
		log.Infof("%s: starting upload of %s", h, f)
		files, err := f.Resolve()
		if err != nil {
			return err
		}
		resolved = append(resolved, files...)
	}

	dirs := make(map[string]struct{})

	for _, f := range resolved {
		destdir, _, err := f.Destination()
		if err != nil {
			return err
		}
		dirs[destdir] = struct{}{}
	}

	for d := range dirs {
		if h.Configurer.FileExist(h, d) {
			continue
		}

		log.Infof("%s: creating directory %s", h, d)
		if err := h.Configurer.MkDir(h, d, exec.Sudo(h)); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}

		if err := h.Configurer.Chmod(h, d, "0755", exec.Sudo(h)); err != nil {
			return fmt.Errorf("failed to set permissions for directory %s: %w", d, err)
		}
	}

	for _, f := range resolved {
		destdir, destfile, err := f.Destination()
		dest := path.Join(destdir, destfile)
		if err != nil {
			return err
		}

		if f.IsURL() {
			err = p.uploadURL(h, f, dest)
		} else {
			err = p.uploadLocal(h, f, dest)
		}

		if err != nil {
			return err
		}

	}

	return nil
}

func (p *UploadFiles) uploadLocal(h *cluster.Host, f cluster.UploadFile, dest string) error {
	log.Infof("%s: uploading %s", h, f)
	if err := h.Upload(f.Source, dest); err != nil {
		return err
	}

	perm := f.PermString
	if perm == "" {
		perm = f.LocalPermString
	}
	return h.Configurer.Chmod(h, dest, perm, exec.Sudo(h))
}

func (p *UploadFiles) uploadURL(h *cluster.Host, f cluster.UploadFile, dest string) error {
	log.Infof("%s: downloading %s", h, f)
	if err := h.Configurer.DownloadURL(h, f.Source, dest); err != nil {
		return err
	}

	if f.PermString == "" {
		return nil
	}

	return h.Configurer.Chmod(h, dest, f.PermString, exec.Sudo(h))
}
