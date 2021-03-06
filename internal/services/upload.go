package services

import (
	"fmt"
	"log"
	"os"

	"github.com/NorthfieldIT/yaml2confluence/internal/confluence"
	"github.com/NorthfieldIT/yaml2confluence/internal/constants"
	"github.com/NorthfieldIT/yaml2confluence/internal/resources"
	"github.com/NorthfieldIT/yaml2confluence/internal/utils"
)

var CHANGE_VERBS = map[resources.ChangeType]string{
	resources.CREATE: "Created",
	resources.UPDATE: "Updated",
	resources.DELETE: "Deleted",
	resources.NOOP:   "Skipped",
}

type IUploadSrv interface {
	UploadSingleResource(string)
	UploadSpace(string)
}

type UploadSrv struct {
	renderSrv IRenderSrv
}

func NewUploadService() UploadSrv {
	return UploadSrv{NewRenderService()}
}

func (us UploadSrv) UploadSingleResource(file string) {
	// dirProps := confluence.GetDirectoryProperties(file)
	// title, markup := us.renderSrv.RenderSingleResource(file)

	// confluence.CreatePage(title, markup, dirProps.SpaceKey, confluence.LoadConfig(dirProps.ConfigPath))
}

func (us UploadSrv) UploadSpace(spaceDirectory string) {
	dirProps := utils.GetDirectoryProperties(spaceDirectory)
	config := confluence.LoadConfig(dirProps.ConfigPath)
	api := confluence.NewConfluenceApiService(dirProps.SpaceKey, config)

	yr := resources.LoadYamlResources(dirProps.SpaceDir)

	if err := resources.EnsureUniqueTitles(yr); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	pt := resources.NewPageTree(yr, resources.GetAnchor(spaceDirectory))

	resources.NewRenderTools(dirProps, true).RenderAll(pt)
	spaceExisted, id, err := api.CreateSpaceIfNotExists()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if !pt.HasAnchor() {
		pt.SetAnchor(id)
	}

	if spaceExisted {
		pages, base, err := api.GetManagedContent()
		if err != nil {
			fmt.Printf("Failed to retrieve managed content from %s space\n%s\n", dirProps.SpaceKey, err.Error())
			os.Exit(1)
		}
		pt.AddRemotes(toRemoteResource(pages, base))
	}

	update(api, pt.GetChanges())
}

func toRemoteResource(pages []confluence.ConfluencePageExpanded, base string) []*resources.RemoteResource {
	remotes := []*resources.RemoteResource{}

	for _, page := range pages {
		ancestors := []resources.Ancestor{}

		for _, ancestor := range page.Ancestors {
			ancestors = append(ancestors, resources.Ancestor{Id: ancestor.Id, Title: ancestor.Title})
		}

		labels := []string{}

		for _, label := range page.Metadata.Labels.Results {
			labels = append(labels, label.Name)
		}

		remotes = append(remotes, &resources.RemoteResource{
			Id:        page.Id,
			Title:     page.Title,
			Labels:    labels,
			Link:      base + page.Links.Webui,
			Version:   page.Version.Number,
			Ancestors: ancestors,
			Sha256: resources.RemoteSha256{
				Id:      page.Metadata.Properties.Sha256.Id,
				Value:   page.Metadata.Properties.Sha256.Value,
				Version: page.Metadata.Properties.Sha256.Version.Number,
			},
		})
	}

	return remotes
}

func update(api confluence.ConfluenceApiService, changes [][]resources.PageUpdate) error {
	for _, group := range changes {
		utils.EachLimit(len(group), 10, func(index int) {
			change := group[index]
			page := change.Page

			switch change.Operation {
			case resources.CREATE, resources.UPDATE:
				id, link, err := api.UpsertPage(page)
				if err != nil {
					log.Fatal(err)
				}
				if change.Operation == resources.CREATE {
					page.Remote = &resources.RemoteResource{Id: id, Link: link}
				}

				extraCalls := []func(){}

				if change.Operation == resources.CREATE || page.Sha256Differs() {
					extraCalls = append(extraCalls, func() {
						err = api.UpsertProperty(page.GetSha256Property())
						if err != nil {
							log.Fatal(err)
						}
					})
				}

				if api.IsServerInstance() && (change.Operation == resources.CREATE || page.LabelsDiffer()) {
					extraCalls = append(extraCalls, func() {
						err = api.SetLabels(id, append([]string{constants.GENERATED_BY_LABEL}, page.GetLabels()...))
						if err != nil {
							log.Fatal(err)
						}
					})
				}

				utils.EachLimit(len(extraCalls), 2, func(index int) { extraCalls[index]() })

				op := CHANGE_VERBS[change.Operation]
				if change.Operation == resources.UPDATE && !page.Sha256Differs() && page.LabelsDiffer() {
					op = "Labels "
				}
				fmt.Printf("%s  %s\n", op, page.Remote.Link)
			case resources.DELETE:
				err := api.DeletePage(page.GetRemoteId())
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%s  %s\n", CHANGE_VERBS[change.Operation], page.Remote.Link)
			case resources.NOOP:
				fmt.Printf("%s  %s\n", CHANGE_VERBS[change.Operation], page.Remote.Link)
			}
		})
	}
	return nil
}
