// Tool to convert a TMX file to an image.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/madjake/go-tiled"
	"github.com/madjake/go-tiled/render"
)

func main() {
	flag.Parse()

	filename := flag.Arg(0)
	img := flag.Arg(1)
	if img == "" {
		img = "map.png"
	}

	m, err := tiled.LoadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	rend, err := render.NewRenderer(m)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = rend.RenderVisibleLayers(); err != nil {
		fmt.Println(err)
		return
	}
	// rend.RenderLayer(1)

	w, err := os.Create(img)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer w.Close()
	if err := rend.SaveAsPng(w); err != nil {
		fmt.Println(err)
		return
	}
}
