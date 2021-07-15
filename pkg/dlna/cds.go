package dlna

// from https://github.com/rclone/rclone
// Copyright (C) 2012 by Nick Craig-Wood http://www.craig-wood.com/nick/

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/dms/upnp"
	"github.com/anacrolix/dms/upnpav"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

var pageSize = 100

type browse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int
	RequestedCount int
}

type contentDirectoryService struct {
	*Server
	upnp.Eventing
	txnManager models.TransactionManager
}

func formatDurationSexagesimal(d time.Duration) string {
	ns := d % time.Second
	d /= time.Second
	s := d % 60
	d /= 60
	m := d % 60
	d /= 60
	h := d
	ret := fmt.Sprintf("%d:%02d:%02d.%09d", h, m, s, ns)
	ret = strings.TrimRight(ret, "0")
	ret = strings.TrimRight(ret, ".")
	return ret
}

func (me *contentDirectoryService) updateIDString() string {
	return fmt.Sprintf("%d", uint32(os.Getpid()))
}

func sceneToContainer(scene *models.Scene, parent string, host string) interface{} {
	// make stash server URL
	// TODO - fix this
	iconURI := (&url.URL{
		Scheme: "http",
		Host:   host,
		Path:   iconPath,
		RawQuery: url.Values{
			"scene": {strconv.Itoa(scene.ID)},
			"c":     {"jpeg"},
		}.Encode(),
	}).String()

	// Object goes first
	obj := upnpav.Object{
		ID:          strconv.Itoa(scene.ID),
		Restricted:  1,
		ParentID:    parent,
		Title:       scene.GetTitle(),
		Class:       "object.item.videoItem",
		Icon:        iconURI,
		AlbumArtURI: iconURI,
	}

	// Wrap up
	item := upnpav.Item{
		Object: obj,
		Res:    make([]upnpav.Resource, 0, 1),
	}

	mimeType := "video/mp4"
	size, _ := strconv.Atoi(scene.Size.String)

	duration := int64(scene.Duration.Float64)

	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   resPath,
			RawQuery: url.Values{
				"scene": {strconv.Itoa(scene.ID)},
			}.Encode(),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Bitrate: uint(scene.Bitrate.Int64),
		// TODO - make %d:%02d:%02d string
		Duration: formatDurationSexagesimal(time.Duration(duration) * time.Second),
		Size:     uint64(size),
		// Resolution: resolution,
	})

	item.Res = append(item.Res, upnpav.Resource{
		URL:          iconURI,
		ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_MED",
	})

	return item
}

// ContentDirectory object from ObjectID.
func (me *contentDirectoryService) objectFromID(id string) (o object, err error) {
	o.Path, err = url.QueryUnescape(id)
	if err != nil {
		return
	}
	if o.Path == "0" {
		o.Path = "/"
	}
	// o.Path = path.Clean(o.Path)
	// if !path.IsAbs(o.Path) {
	// 	err = fmt.Errorf("bad ObjectID %v", o.Path)
	// 	return
	// }
	o.RootObjectPath = me.RootObjectPath

	return
}

func childPath(paths []string) []string {
	if len(paths) > 1 {
		return paths[1:]
	}

	return nil
}

func (me *contentDirectoryService) Handle(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	host := r.Host
	// userAgent := r.UserAgent()
	switch action {
	case "GetSystemUpdateID":
		return map[string]string{
			"Id": me.updateIDString(),
		}, nil
	case "GetSortCapabilities":
		return map[string]string{
			"SortCaps": "dc:title",
		}, nil
	case "Browse":
		var browse browse
		if err := xml.Unmarshal([]byte(argsXML), &browse); err != nil {
			return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "cannot unmarshal browse argument: %s", err.Error())
		}

		obj, err := me.objectFromID(browse.ObjectID)
		if err != nil {
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, err.Error())
		}

		switch browse.BrowseFlag {
		case "BrowseDirectChildren":
			return me.handleBrowseDirectChildren(obj, host)
		case "BrowseMetadata":
			return me.handleBrowseMetadata(obj, host)
		default:
			return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "unhandled browse flag: %v", browse.BrowseFlag)
		}
	case "GetSearchCapabilities":
		return map[string]string{
			"SearchCaps": "",
		}, nil
	// from https://github.com/rclone/rclone/blob/master/cmd/serve/dlna/cds.go
	// Samsung Extensions
	case "X_GetFeatureList":
		return map[string]string{
			"FeatureList": `<Features xmlns="urn:schemas-upnp-org:av:avs" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">
	<Feature name="samsung.com_BASICVIEW" version="1">
		<container id="0" type="object.item.imageItem"/>
		<container id="0" type="object.item.audioItem"/>
		<container id="0" type="object.item.videoItem"/>
	</Feature>
	</Features>`}, nil
	case "X_SetBookmark":
		// just ignore
		return map[string]string{}, nil
	default:
		return nil, upnp.InvalidActionError
	}
}

func (me *contentDirectoryService) handleBrowseDirectChildren(obj object, host string) (map[string]string, error) {
	// Read folder and return children
	// TODO: check if obj == 0 and return root objects
	// TODO: check if special path and return files

	var objs []interface{}

	if obj.IsRoot() {
		objs = getRootObjects()
	}

	paths := strings.Split(obj.Path, "/")

	// All videos
	if obj.Path == "all" {
		objs = me.getAllScenes(host)
	}

	if strings.HasPrefix(obj.Path, "all/") {
		page := getPageFromID(paths)
		if page != nil {
			objs = me.getPageVideos(&models.SceneFilterType{}, "all", *page, host)
		}
	}

	// Saved searches
	// if obj.Path == "saved-searches" {
	// 	var savedPlaylists []models.Playlist
	// 	db, _ := models.GetDB()
	// 	db.Where("is_deo_enabled = ?", true).Order("ordering asc").Find(&savedPlaylists)
	// 	db.Close()

	// 	for _, playlist := range savedPlaylists {
	// 		objs = append(objs, upnpav.Container{Object: upnpav.Object{
	// 			ID:         "saved-searches/" + strconv.Itoa(int(playlist.ID)),
	// 			Restricted: 1,
	// 			ParentID:   "saved-searches",
	// 			Class:      "object.container.storageFolder",
	// 			Title:      playlist.Name,
	// 		}})
	// 	}
	// }

	// if strings.HasPrefix(obj.Path, "saved-searches/") {
	// 	id := strings.Split(obj.Path, "/")

	// 	var savedPlaylist models.Playlist
	// 	db, _ := models.GetDB()
	// 	db.Where("id = ?", id[1]).First(&savedPlaylist)
	// 	db.Close()

	// 	var r models.RequestSceneList
	// 	if err := json.Unmarshal([]byte(savedPlaylist.SearchParams), &r); err == nil {
	// 		r.IsAccessible = optional.NewBool(true)
	// 		r.IsAvailable = optional.NewBool(true)
	// 		data := models.QueryScenesFull(r)

	// 		for i := range data.Scenes {
	// 			objs = append(objs, me.sceneToContainer(data.Scenes[i], "sites/"+id[1], host))
	// 		}
	// 	}
	// }

	// Studios
	if obj.Path == "studios" {
		objs = me.getStudios()
	}

	if strings.HasPrefix(obj.Path, "studios/") {
		objs = me.getStudioScenes(childPath(paths), host)
	}

	// Tags
	if obj.Path == "tags" {
		objs = me.getTags()
	}

	if strings.HasPrefix(obj.Path, "tags/") {
		objs = me.getTagScenes(childPath(paths), host)
	}

	// Performers
	if obj.Path == "performers" {
		objs = me.getPerformers()
	}

	if strings.HasPrefix(obj.Path, "performers/") {
		objs = me.getPerformerScenes(childPath(paths), host)
	}

	// Movies
	if obj.Path == "movies" {
		objs = me.getMovies()
	}

	if strings.HasPrefix(obj.Path, "movies/") {
		objs = me.getMovieScenes(childPath(paths), host)
	}

	// Rating
	if obj.Path == "rating" {
		objs = me.getRating()
	}

	if strings.HasPrefix(obj.Path, "rating/") {
		objs = me.getRatingScenes(childPath(paths), host)
	}

	return makeBrowseResult(objs, me.updateIDString())
}

func (me *contentDirectoryService) handleBrowseMetadata(obj object, host string) (map[string]string, error) {
	var objs []interface{}
	var updateID string

	// if numeric, then must be scene, otherwise handle as if path
	sceneID, err := strconv.Atoi(obj.Path)
	if err != nil {
		// #1465 - handle root object
		if obj.IsRoot() {
			objs = getRootObject()
		} else {
			// HACK: just create a fake storage folder to return. The name won't
			// be correct, but hopefully the names returned from handleBrowseDirectChildren
			// will be used instead.
			objs = []interface{}{makeStorageFolder(obj.ID(), obj.ID(), obj.ParentID())}
		}

		updateID = me.updateIDString()
	} else {
		var scene *models.Scene

		if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
			scene, err = r.Scene().Find(sceneID)
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			logger.Error(err.Error())
		}

		if scene != nil {
			upnpObject := sceneToContainer(scene, "-1", host)
			objs = []interface{}{upnpObject}

			// http://upnp.org/specs/av/UPnP-av-ContentDirectory-v1-Service.pdf
			// maximum update ID is 2**32, then rolls back to 0
			const maxUpdateID int64 = 1 << 32
			updateID = fmt.Sprint(scene.UpdatedAt.Timestamp.Unix() % maxUpdateID)
		} else {
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, "scene not found")
		}
	}

	return makeBrowseResult(objs, updateID)
}

func makeBrowseResult(objs []interface{}, updateID string) (map[string]string, error) {
	result, err := xml.Marshal(objs)
	if err != nil {
		return nil, upnp.Errorf(upnp.ActionFailedErrorCode, "could not marshal objects: %s", err.Error())
	}

	return map[string]string{
		"TotalMatches":   fmt.Sprint(len(objs)),
		"NumberReturned": fmt.Sprint(len(objs)),
		"Result":         didl_lite(string(result)),
		"UpdateID":       updateID,
	}, nil
}

func makeStorageFolder(id, title, parentID string) upnpav.Container {
	defaultChildCount := 1
	return upnpav.Container{
		Object: upnpav.Object{
			ID:         id,
			Restricted: 1,
			ParentID:   parentID,
			Class:      "object.container.storageFolder",
			Title:      title,
		},
		ChildCount: defaultChildCount,
	}
}

func getRootObject() []interface{} {
	const rootID = "0"

	return []interface{}{makeStorageFolder(rootID, "stash", "-1")}
}

func getRootObjects() []interface{} {
	const rootID = "0"

	var objs []interface{}

	objs = append(objs, makeStorageFolder("all", "all", rootID))
	objs = append(objs, makeStorageFolder("performers", "performers", rootID))
	objs = append(objs, makeStorageFolder("tags", "tags", rootID))
	objs = append(objs, makeStorageFolder("studios", "studios", rootID))
	objs = append(objs, makeStorageFolder("movies", "movies", rootID))
	objs = append(objs, makeStorageFolder("rating", "rating", rootID))

	return objs
}

func (me *contentDirectoryService) getVideos(sceneFilter *models.SceneFilterType, parentID string, host string) []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		sort := "title"
		findFilter := &models.FindFilterType{
			PerPage: &pageSize,
			Sort:    &sort,
		}

		scenes, total, err := r.Scene().Query(sceneFilter, findFilter)
		if err != nil {
			return err
		}

		if total > pageSize {
			pager := scenePager{
				sceneFilter: sceneFilter,
				parentID:    parentID,
			}

			objs, err = pager.getPages(r, total)
			if err != nil {
				return err
			}
		} else {
			for _, s := range scenes {
				objs = append(objs, sceneToContainer(s, parentID, host))
			}
		}

		return nil
	}); err != nil {
		logger.Error(err.Error())
	}

	return objs
}

func (me *contentDirectoryService) getPageVideos(sceneFilter *models.SceneFilterType, parentID string, page int, host string) []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		pager := scenePager{
			sceneFilter: sceneFilter,
			parentID:    parentID,
		}

		var err error
		objs, err = pager.getPageVideos(r, page, host)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		logger.Error(err.Error())
	}

	return objs
}

func getPageFromID(paths []string) *int {
	i := utils.StrIndex(paths, "page")
	if i == -1 || i+1 >= len(paths) {
		return nil
	}

	ret, err := strconv.Atoi(paths[i+1])
	if err != nil {
		return nil
	}

	return &ret
}

func (me *contentDirectoryService) getAllScenes(host string) []interface{} {
	return me.getVideos(&models.SceneFilterType{}, "all", host)
}

func (me *contentDirectoryService) getStudios() []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		studios, err := r.Studio().All()
		if err != nil {
			return err
		}

		for _, s := range studios {
			objs = append(objs, makeStorageFolder("studios/"+strconv.Itoa(s.ID), s.Name.String, "studios"))
		}

		return nil
	}); err != nil {
		logger.Errorf(err.Error())
	}

	return objs
}

func (me *contentDirectoryService) getStudioScenes(paths []string, host string) []interface{} {
	sceneFilter := &models.SceneFilterType{
		Studios: &models.HierarchicalMultiCriterionInput{
			Modifier: models.CriterionModifierIncludes,
			Value:    []string{paths[0]},
			Depth:    0,
		},
	}

	parentID := "studios/" + strings.Join(paths, "/")

	page := getPageFromID(paths)
	if page != nil {
		return me.getPageVideos(sceneFilter, parentID, *page, host)
	}

	return me.getVideos(sceneFilter, parentID, host)
}

func (me *contentDirectoryService) getTags() []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		tags, err := r.Tag().All()
		if err != nil {
			return err
		}

		for _, s := range tags {
			objs = append(objs, makeStorageFolder("tags/"+strconv.Itoa(s.ID), s.Name, "tags"))
		}

		return nil
	}); err != nil {
		logger.Errorf(err.Error())
	}

	return objs
}

func (me *contentDirectoryService) getTagScenes(paths []string, host string) []interface{} {
	sceneFilter := &models.SceneFilterType{
		Tags: &models.MultiCriterionInput{
			Modifier: models.CriterionModifierIncludes,
			Value:    []string{paths[0]},
		},
	}

	parentID := "tags/" + strings.Join(paths, "/")

	page := getPageFromID(paths)
	if page != nil {
		return me.getPageVideos(sceneFilter, parentID, *page, host)
	}

	return me.getVideos(sceneFilter, parentID, host)
}

func (me *contentDirectoryService) getPerformers() []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		performers, err := r.Performer().All()
		if err != nil {
			return err
		}

		for _, s := range performers {
			objs = append(objs, makeStorageFolder("performers/"+strconv.Itoa(s.ID), s.Name.String, "performers"))
		}

		return nil
	}); err != nil {
		logger.Errorf(err.Error())
	}

	return objs
}

func (me *contentDirectoryService) getPerformerScenes(paths []string, host string) []interface{} {
	sceneFilter := &models.SceneFilterType{
		Performers: &models.MultiCriterionInput{
			Modifier: models.CriterionModifierIncludes,
			Value:    []string{paths[0]},
		},
	}

	parentID := "performers/" + strings.Join(paths, "/")

	page := getPageFromID(paths)
	if page != nil {
		return me.getPageVideos(sceneFilter, parentID, *page, host)
	}

	return me.getVideos(sceneFilter, parentID, host)
}

func (me *contentDirectoryService) getMovies() []interface{} {
	var objs []interface{}

	if err := me.txnManager.WithReadTxn(context.TODO(), func(r models.ReaderRepository) error {
		movies, err := r.Movie().All()
		if err != nil {
			return err
		}

		for _, s := range movies {
			objs = append(objs, makeStorageFolder("movies/"+strconv.Itoa(s.ID), s.Name.String, "movies"))
		}

		return nil
	}); err != nil {
		logger.Errorf(err.Error())
	}

	return objs
}

func (me *contentDirectoryService) getMovieScenes(paths []string, host string) []interface{} {
	sceneFilter := &models.SceneFilterType{
		Movies: &models.MultiCriterionInput{
			Modifier: models.CriterionModifierIncludes,
			Value:    []string{paths[0]},
		},
	}

	parentID := "movies/" + strings.Join(paths, "/")

	page := getPageFromID(paths)
	if page != nil {
		return me.getPageVideos(sceneFilter, parentID, *page, host)
	}

	return me.getVideos(sceneFilter, parentID, host)
}

func (me *contentDirectoryService) getRating() []interface{} {
	var objs []interface{}

	for r := 1; r <= 5; r++ {
		rStr := strconv.Itoa(r)
		objs = append(objs, makeStorageFolder("rating/"+rStr, rStr, "rating"))
	}

	return objs
}

func (me *contentDirectoryService) getRatingScenes(paths []string, host string) []interface{} {
	r, err := strconv.Atoi(paths[0])
	if err != nil {
		return nil
	}

	sceneFilter := &models.SceneFilterType{
		Rating: &models.IntCriterionInput{
			Modifier: models.CriterionModifierEquals,
			Exact:    r,
		},
	}

	parentID := "rating/" + strings.Join(paths, "/")

	page := getPageFromID(paths)
	if page != nil {
		return me.getPageVideos(sceneFilter, parentID, *page, host)
	}

	return me.getVideos(sceneFilter, parentID, host)
}

// Represents a ContentDirectory object.
type object struct {
	Path           string // The cleaned, absolute path for the object relative to the server.
	RootObjectPath string
}

// Returns the actual local filesystem path for the object.
func (o *object) FilePath() string {
	return filepath.Join(o.RootObjectPath, filepath.FromSlash(o.Path))
}

// Returns the ObjectID for the object. This is used in various ContentDirectory actions.
func (o object) ID() string {
	if len(o.Path) == 1 {
		return "0"
	}
	return url.QueryEscape(o.Path)
}

func (o *object) IsRoot() bool {
	return o.Path == "/"
}

// Returns the object's parent ObjectID. Fortunately it can be deduced from the
// ObjectID (for now).
func (o object) ParentID() string {
	if o.IsRoot() {
		return "-1"
	}
	o.Path = path.Dir(o.Path)
	return o.ID()
}
