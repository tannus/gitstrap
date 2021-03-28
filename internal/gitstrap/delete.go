package gitstrap

import (
	"errors"
	"fmt"

	"github.com/g4s8/gitstrap/internal/spec"
	"github.com/google/go-github/v33/github"
)

func (g *Gitstrap) Delete(m *spec.Model) error {
	switch m.Kind {
	case spec.KindRepo:
		return g.deleteRepo(m)
	case spec.KindReadme:
		return g.deleteReadme(m)
	case spec.KindHook:
		return g.deleteHook(m)
	default:
		return &errUnsupportModelKind{m.Kind}
	}
}

func (g *Gitstrap) deleteRepo(m *spec.Model) error {
	ctx, cancel := g.newContext()
	defer cancel()
	meta := m.Metadata
	owner := meta.Owner
	if owner == "" {
		owner = g.me
	}
	if _, err := g.gh.Repositories.Delete(ctx, owner, meta.Name); err != nil {
		return err
	}
	return nil
}

type errReadmeNotExists struct {
	owner, repo string
}

func (e *errReadmeNotExists) Error() string {
	return fmt.Sprintf("README `%s/%s` doesn't exist", e.owner, e.repo)
}

func (g *Gitstrap) deleteReadme(m *spec.Model) error {
	ctx, cancel := g.newContext()
	defer cancel()
	spec := new(spec.Readme)
	meta := m.Metadata
	if err := m.GetSpec(spec); err != nil {
		return err
	}
	owner := m.Metadata.Owner
	if owner == "" {
		owner = g.me
	}
	repo, _, err := g.gh.Repositories.Get(ctx, owner, spec.Selector.Repository)
	if err != nil {
		return err
	}
	msg := "README.md removed"
	if cm, ok := meta.Annotations["commitMessage"]; ok {
		msg = cm
	}
	opts := &github.RepositoryContentFileOptions{
		Message: &msg,
	}
	getopts := new(github.RepositoryContentGetOptions)
	cnt, _, rsp, err := g.gh.Repositories.GetContents(ctx, owner, repo.GetName(), "README.md", getopts)
	if rsp.StatusCode == 404 {
		return &errReadmeNotExists{owner, repo.GetName()}
	}
	if err != nil {
		return err
	}
	if *cnt.Type != "file" {
		return &errReadmeNotFile{*cnt.Type}
	}
	opts.SHA = cnt.SHA
	if _, _, err := g.gh.Repositories.DeleteFile(ctx, owner, repo.GetName(), "README.md", opts); err != nil {
		return err
	}
	return nil
}

var errHookIdRequired = errors.New("Hook metadata ID required")

func (g *Gitstrap) deleteHook(m *spec.Model) error {
	ctx, cancel := g.newContext()
	defer cancel()
	hook := new(spec.Hook)
	if err := m.GetSpec(hook); err != nil {
		return err
	}
	owner := g.getOwner(m)
	if m.Metadata.ID == nil {
		return errHookIdRequired
	}
	id := *m.Metadata.ID
	if hook.Selector.Repository != "" {
		_, err := g.gh.Repositories.DeleteHook(ctx, owner, hook.Selector.Repository, id)
		return err
	} else if hook.Selector.Organization != "" {
		_, err := g.gh.Organizations.DeleteHook(ctx, hook.Selector.Organization, id)
		return err
	} else {
		return errHookSelectorEmpty
	}
}