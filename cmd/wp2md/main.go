package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/JohannesKaufmann/html-to-markdown"
	"github.com/alexflint/go-arg"
	"github.com/blockloop/scan"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"html"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	postTemplate = `---
title: {{ .Title }}
date: {{ .Created }}
draft: {{ .Draft }}
slug: {{ .Name }}
---

{{ .Content }}
`
)

type Post struct {
	Author  string
	Title   string
	Name    string
	Content string
	Created time.Time
	Status  string
	Draft   bool
}

func main() {
	tmpl, err := template.New("hugoposts").Parse(postTemplate)
	if err != nil {
		log.WithError(err).Fatal("error parsing post template")
	}

	var args struct {
		DBFilename       string `arg:"positional,required"`
		ContentDirectory string `arg:"env:CONTENT_DIRECTORY" default:"posts"`
	}
	arg.MustParse(&args)

	log.Infof("reading %s", args.DBFilename)

	sqliteDatabase, err := sql.Open("sqlite3", args.DBFilename)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"filename": args.DBFilename}).Fatal("error opening database")
	}
	defer sqliteDatabase.Close()

	rows, err := sqliteDatabase.Query(`select u.display_name,
	p.post_title as title,
	p.post_name as name,
	p.post_content as content,
	p.post_date as created,
	p.post_status as status,
	p.post_status == 'draft' as draft
	from wp_posts p
	inner join wp_users u on (p.post_author = u.ID)
	where p.post_status in ("draft", "publish")
	order by p.post_date ASC`)
	if err != nil {
		log.WithError(err).Fatal("error querying sqlite database")
	}
	defer rows.Close()

	var posts []Post
	if err := scan.Rows(&posts, rows); err != nil {
		log.WithError(err).Fatal("error scanning rows")
	}

	mdConverter := md.NewConverter("", true, nil)

	log.Infof("processing %d posts", len(posts))

	for _, post := range posts {
		content, err := mdConverter.ConvertString(post.Content)
		if err != nil {
			log.WithError(err).Error("error converting post content html to markdown")
			continue
		}
		post.Content = content

		post.Title = html.EscapeString(post.Title)
		post.Title = strings.Replace(post.Title, ":", "%3A", -1)

		var renderedPost bytes.Buffer
		if err := tmpl.Execute(&renderedPost, post); err != nil {
			log.WithError(err).Errorf("error executing template for post %s", post.Name)
			continue
		}

		filename := filepath.Join(args.ContentDirectory, fmt.Sprintf("%s.md", post.Name))
		if err := os.WriteFile(filename, renderedPost.Bytes(), 0644); err != nil {
			log.WithError(err).Errorf("error writing file for post %s", post.Name)
			continue
		}
		log.Infof("wrote %s", filename)
	}
}
