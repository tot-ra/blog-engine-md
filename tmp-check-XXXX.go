package main

import (
  "fmt"
  "path/filepath"
  "github.com/tot-ra/blog-engine/internal/builder"
  "github.com/tot-ra/blog-engine/internal/config"
)

func main() {
  cfg, err := config.Load("../dina.kurapov.ee/config.yaml")
  if err != nil { panic(err) }
  old, _ := filepath.Abs("../dina.kurapov.ee")
  _ = old
  index, err := builder.Discover("../dina.kurapov.ee/content")
  if err != nil { panic(err) }
  langs := map[string]struct{}{"rus":{},"est":{}}
  pb := builder.NewPageBuilder(cfg.Site.URL, cfg.I18n.Default, langs)
  pages, errs := builder.BuildPagesForTests(pb, index.MarkdownFiles)
  if len(errs) > 0 { fmt.Println("errs", len(errs)) }
  m := map[string]*builder.Page{}
  for _, p := range pages { m[p.ID] = p }
  tree := builder.NewNavigationBuilder().BuildTree(m)
  node := tree.ByPath["/est/study/EG/"]
  if node == nil { panic("missing EG node") }
  for _, c := range node.Children { fmt.Printf("%q order=%d type=%s children=%d url=%s\n", c.Title, c.Order, c.Type, len(c.Children), c.URL) }
}
