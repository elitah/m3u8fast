package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/grafov/m3u8"
)

var (
	//
	httpClient *http.Client
)

func urlParse(s string) (*url.URL, error) {
	//
	if u, err := url.Parse(s); nil == err {
		//
		if "http" == u.Scheme || "https" == u.Scheme {
			//
			return u, nil
		} else {
			//
			return nil, syscall.EINVAL
		}
	} else {
		//
		return nil, err
	}
}

func read_m3u8_from_url(url string) (m3u8.Playlist, m3u8.ListType, error) {
	//
	if u, err := urlParse(url); nil == err {
		//
		if resp, err := httpClient.Get(u.String()); nil == err {
			//
			defer resp.Body.Close()
			//
			return m3u8.DecodeFrom(resp.Body, true)
		} else {
			//
			return nil, 0, err
		}
	} else {
		//
		return nil, 0, err
	}
}

func main() {
	//
	var count int
	//
	var force bool
	//
	var wg sync.WaitGroup
	//
	var urls []string
	//
	flag.IntVar(&count, "n", 1, "count")
	flag.BoolVar(&force, "f", false, "force")
	//
	flag.Parse()
	//
	addURL := func(url string) {
		//
		fmt.Println("add", url)
		//
		for _, item := range urls {
			//
			if url == item {
				//
				return
			}
		}
		//
		urls = append(urls, url)
	}
	//
	httpClient = &http.Client{
		//
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	//
	for _, item := range flag.Args() {
		//
		if p, listType, err := read_m3u8_from_url(item); nil == err {
			//
			if m3u8.MASTER == listType {
				//
				if list, ok := p.(*m3u8.MasterPlaylist); ok {
					//
					for _, _item := range list.Variants {
						//
						if u, err := urlParse(_item.URI); nil == err {
							//
							addURL(u.String())
						} else if u, err := urlParse(item); nil == err {
							//
							u.Path = _item.URI
							//
							addURL(u.String())
						}
					}
				}
			} else {
				//
				addURL(item)
			}
		} else {
			//
			fmt.Println(err)
		}
	}
	//
	if 1 > count {
		//
		count = 1
	}
	//
	limit := make(chan int, count)
	//
	for _, item := range urls {
		//
		if u, err := urlParse(item); nil == err {
			//
			if p, listType, err := read_m3u8_from_url(u.String()); nil == err {
				//
				if m3u8.MEDIA == listType {
					//
					if list, ok := p.(*m3u8.MediaPlaylist); ok {
						//
						for _, item := range list.Segments {
							//
							if nil != item {
								//
								u.Path = item.URI
								//
								item.URI = item.URI[1:]
								//
								if dir := filepath.Dir(u.Path); "" != dir {
									//
									os.MkdirAll(dir[1:], 0755)
								}
								//
								wg.Add(1)
								//
								go func(url, filepath string) {
									//
									defer func() {
										//
										<-limit
										//
										wg.Done()
									}()
									//
									limit <- 1
									//
									start := time.Now()
									//
									if f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644); nil == err {
										//
										length, _ := f.Seek(0, os.SEEK_END)
										//
										f.Seek(0, os.SEEK_SET)
										//
										if resp, err := httpClient.Get(url); nil == err {
											//
											defer resp.Body.Close()
											//
											if http.StatusOK == resp.StatusCode {
												//
												if force || length != resp.ContentLength {
													//
													f.Truncate(0)
													//
													f.ReadFrom(resp.Body)
												}
											} else {
												//
												fmt.Println(url, resp.StatusCode)
											}
										}
										//
										f.Close()
									}
									//
									fmt.Println(time.Since(start))
								}(u.String(), u.Path[1:])
							}
						}
						//
						if f, err := os.OpenFile(fmt.Sprintf("index_%d.m3u8", time.Now().Unix()), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); nil == err {
							//
							if b := list.Encode(); nil != b {
								//
								f.Truncate(0)
								//
								f.ReadFrom(b)
							}
							//
							f.Close()
						}
					}
				}
			} else {
				//
				fmt.Println(err)
			}
		} else {
			//
			fmt.Println(err)
		}
	}
	//
	wg.Wait()
}
