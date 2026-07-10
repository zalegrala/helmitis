package stamp

import (
	"sort"

	"github.com/zalegrala/chartwright/interchange"
)

// File is one output file destined for the chart directory.
type File struct {
	Path    string
	Content []byte
}

// Build turns a validated interchange Document into the full set of chart files.
// Output is deterministic: stable file ordering and stable content.
func Build(doc interchange.Document) ([]File, error) {
	doc, err := Lower(doc)
	if err != nil {
		return nil, err
	}
	var files []File

	for _, r := range doc.Resources {
		content, err := renderResource(r)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: r.File, Content: []byte(content)})
	}

	vals, err := buildValues(doc)
	if err != nil {
		return nil, err
	}
	valsYAML, err := marshalValues(vals)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Path: "values.yaml", Content: valsYAML})

	schema, err := buildValuesSchema(doc)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Path: "values.schema.json", Content: schema})

	chartFile, err := chartYAML(doc.Chart)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Path: "Chart.yaml", Content: chartFile})

	files = append(files, File{Path: "templates/_helpers.tpl", Content: []byte(helpersTpl)})

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
