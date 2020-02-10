package controllers

import (
	"fmt"
)

func CreateLinks(path string, offset, limit, total int, otherParams ...string) Links {
	var queryStr string

	for _, param := range otherParams {
		if len(param) > 0 {
			queryStr = fmt.Sprintf("%v&%v", queryStr, param)
		}
	}

	pager := pager{path, offset, limit, total, queryStr}
	links := Links{
		First:    pager.createLink(0),
		Last:     pager.createLastLink(),
		Next:     pager.createNextLink(),
		Previous: pager.createPreviousLink(),
	}

	return links
}

type pager struct {
	path        string
	offset      int
	limit       int
	total       int
	otherParams string
}

func (p pager) createLink(linkOffset int) string {
	link := fmt.Sprintf("%s?offset=%d&limit=%d%s",
		p.path, linkOffset, p.limit, p.otherParams)
	return link
}

func (p pager) createLastLink() string {
	lastOffset := ((p.total / p.limit) - 1) * p.limit
	if lastOffset < 0 {
		lastOffset = 0
	}

	return p.createLink(lastOffset)
}

func (p pager) createNextLink() *string {
	if p.total <= p.offset+p.limit {
		return nil
	}

	next := p.createLink(p.offset + p.limit)
	return &next
}

func (p pager) createPreviousLink() *string {
	if p.offset == 0 {
		return nil
	}
	curPage := p.offset / p.limit
	prevOffset := 0
	if curPage > 0 {
		prevOffset = (curPage - 1) * p.limit
	}
	link := p.createLink(prevOffset)
	return &link
}
