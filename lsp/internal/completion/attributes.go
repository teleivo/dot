package completion

import (
	"cmp"
	"slices"
	"strings"
)

// AttrType identifies an attribute's value type.
// See the [Graphviz attribute type documentation] for the full reference.
//
// [Graphviz attribute type documentation]: https://graphviz.org/docs/attr-types/
type AttrType int

const (
	TypeUnknown   AttrType = iota
	TypeArrowType
	TypeDirType
	TypeLayout
)

// attrTypeInfo holds metadata for each AttrType, indexed by the type value.
var attrTypeInfo = [...]struct {
	// Name is the type name as used in Graphviz documentation (e.g., "dirType").
	Name string
	// Values contains valid values for this type. May not be exhaustive for
	// complex types like arrowType where values can be combined.
	Values []string
	// Documentation is a brief description of what the type represents.
	Documentation string
}{
	TypeUnknown:   {"", nil, ""},
	TypeArrowType: {"arrowType", []string{"box", "crow", "curve", "diamond", "dot", "icurve", "inv", "none", "normal", "tee", "vee"}, "Arrow shape"},
	TypeDirType:   {"dirType", []string{"forward", "back", "both", "none"}, "Edge arrow direction"},
	TypeLayout:    {"layout", []string{"dot", "neato", "twopi", "circo", "fdp", "sfdp", "patchwork", "osage"}, "Layout engine name"},
}

// String returns the type name (e.g., "dirType").
func (t AttrType) String() string { return attrTypeInfo[t].Name }

// Values returns the valid values for this type.
func (t AttrType) Values() []string { return attrTypeInfo[t].Values }

// Documentation returns a brief description of the type.
func (t AttrType) Documentation() string { return attrTypeInfo[t].Documentation }

// URL returns the Graphviz documentation URL for this type.
func (t AttrType) URL() string {
	if t == TypeUnknown {
		return ""
	}
	return "https://graphviz.org/docs/attr-types/" + t.String() + "/"
}

// AttributeContext represents which DOT elements an attribute can be applied to.
// These correspond to the "Used By" column in the [Graphviz attribute documentation]:
//   - Graph (G): graph-level attributes, e.g., graph [rankdir=LR]
//   - Subgraph (S): subgraph attributes
//   - Cluster (C): cluster subgraph attributes (subgraph with ID starting with "cluster_")
//   - Node (N): node attributes, e.g., a [shape=box]
//   - Edge (E): edge attributes, e.g., a -> b [style=dashed]
//
// [Graphviz attribute documentation]: https://graphviz.org/doc/info/attrs.html
type AttributeContext uint

const (
	Graph    AttributeContext = 1 << iota // Graph-level attributes (e.g., rankdir, splines)
	Subgraph                              // Subgraph attributes (e.g., rank)
	Cluster                               // Cluster subgraph attributes (subgraph with "cluster_" prefix)
	Node                                  // Node attributes (e.g., shape, label)
	Edge                                  // Edge attributes (e.g., arrowhead, style)
)

// String returns the string representation of the attribute context.
// For combined contexts (bitmask), it returns a comma-separated list.
func (c AttributeContext) String() string {
	if c == 0 {
		return ""
	}

	// Pre-allocate for all context kinds
	contexts := make([]AttributeContext, 0, 5)
	for remaining := c; remaining != 0; {
		bit := remaining & -remaining
		contexts = append(contexts, bit)
		remaining &^= bit
	}

	var result strings.Builder
	for i, ctx := range contexts {
		if i > 0 {
			result.WriteString(", ")
		}
		switch ctx {
		case Graph:
			result.WriteString("Graph")
		case Subgraph:
			result.WriteString("Subgraph")
		case Cluster:
			result.WriteString("Cluster")
		case Node:
			result.WriteString("Node")
		case Edge:
			result.WriteString("Edge")
		}
	}
	return result.String()
}

// Attribute represents a Graphviz attribute with its applicable contexts and documentation.
type Attribute struct {
	// Name is the attribute name as used in DOT syntax (e.g., "shape", "label").
	Name string
	// Type is the attribute's value type (e.g., TypeDirType, TypeBool).
	Type AttrType
	// UsedBy indicates which DOT elements this attribute can be applied to.
	// Matches the "Used By" column from the [Graphviz attribute documentation].
	//
	// [Graphviz attribute documentation]: https://graphviz.org/doc/info/attrs.html
	UsedBy AttributeContext
	// Documentation is a brief description of what the attribute does.
	Documentation string
	// MarkdownDoc is the precomputed markdown documentation for LSP completion.
	MarkdownDoc string
}

// URL returns the Graphviz documentation URL for this attribute.
func (a Attribute) URL() string {
	return "https://graphviz.org/docs/attrs/" + a.Name + "/"
}

// markdownDoc generates the markdown documentation for this attribute.
func (a Attribute) markdownDoc() string {
	var sb strings.Builder
	sb.WriteString(a.Documentation)
	sb.WriteString("\n\n")

	if a.Type != TypeUnknown {
		sb.WriteString("**Type:** [")
		sb.WriteString(a.Type.String())
		sb.WriteString("](")
		sb.WriteString(a.Type.URL())
		sb.WriteString(")")

		if values := a.Type.Values(); len(values) > 0 {
			sb.WriteString(": `")
			sb.WriteString(values[0])
			for _, v := range values[1:] {
				sb.WriteString("` | `")
				sb.WriteString(v)
			}
			sb.WriteString("`")
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString("[Docs](")
	sb.WriteString(a.URL())
	sb.WriteString(")")
	return sb.String()
}

// Attributes contains all Graphviz attributes sorted by name.
// See the [Graphviz attribute documentation] for the full reference.
//
// [Graphviz attribute documentation]: https://graphviz.org/doc/info/attrs.html
var Attributes = func() []Attribute {
	attributes := []Attribute{
		{Name: "_background", UsedBy: Graph, Documentation: "Specifies arbitrary background using xdot format strings"},
		{Name: "area", UsedBy: Node | Cluster, Documentation: "referred area for node or empty cluster (patchwork layout)"},
		{Name: "arrowhead", Type: TypeArrowType, UsedBy: Edge, Documentation: "Style of arrowhead on edge head node"},
		{Name: "arrowsize", UsedBy: Edge, Documentation: "Multiplicative scale factor for arrowheads"},
		{Name: "arrowtail", Type: TypeArrowType, UsedBy: Edge, Documentation: "Style of arrowhead on edge tail node"},
		{Name: "bb", UsedBy: Cluster | Graph, Documentation: "Bounding box of drawing in points (write-only)"},
		{Name: "beautify", UsedBy: Graph, Documentation: "Whether to draw leaf nodes in circle around root (sfdp)"},
		{Name: "bgcolor", UsedBy: Graph | Cluster, Documentation: "Canvas background color"},
		{Name: "center", UsedBy: Graph, Documentation: "Whether to center drawing in output canvas"},
		{Name: "charset", UsedBy: Graph, Documentation: "Character encoding for text labels"},
		{Name: "class", UsedBy: Edge | Node | Cluster | Graph, Documentation: "Classnames for SVG element styling"},
		{Name: "cluster", UsedBy: Cluster | Subgraph, Documentation: "Whether subgraph is a cluster"},
		{Name: "clusterrank", UsedBy: Graph, Documentation: "Mode for handling clusters (dot layout)"},
		{Name: "color", UsedBy: Edge | Node | Cluster, Documentation: "Basic drawing color for graphics"},
		{Name: "colorscheme", UsedBy: Edge | Node | Cluster | Graph, Documentation: "Color scheme namespace for interpreting color names"},
		{Name: "comment", UsedBy: Edge | Node | Graph, Documentation: "Comments inserted into output"},
		{Name: "compound", UsedBy: Graph, Documentation: "Allow edges between clusters (dot layout)"},
		{Name: "concentrate", UsedBy: Graph, Documentation: "Use edge concentrators"},
		{Name: "constraint", UsedBy: Edge, Documentation: "Whether edge used in node ranking (dot layout)"},
		{Name: "Damping", UsedBy: Graph, Documentation: "Factor damping force motions (neato layout)"},
		{Name: "decorate", UsedBy: Edge, Documentation: "Connect edge label to edge with line"},
		{Name: "defaultdist", UsedBy: Graph, Documentation: "Distance between nodes in separate components (neato)"},
		{Name: "dim", UsedBy: Graph, Documentation: "Number of dimensions for layout"},
		{Name: "dimen", UsedBy: Graph, Documentation: "Number of dimensions for rendering"},
		{Name: "dir", Type: TypeDirType, UsedBy: Edge, Documentation: "Edge type for drawing arrowheads"},
		{Name: "diredgeconstraints", UsedBy: Graph, Documentation: "Constrain edges to point downwards (neato)"},
		{Name: "distortion", UsedBy: Node, Documentation: "Distortion factor for polygon shapes"},
		{Name: "dpi", UsedBy: Graph, Documentation: "Expected pixels per inch on display device"},
		{Name: "edgehref", UsedBy: Edge, Documentation: "Synonym for edgeURL"},
		{Name: "edgetarget", UsedBy: Edge, Documentation: "Browser window for edgeURL link"},
		{Name: "edgetooltip", UsedBy: Edge, Documentation: "Tooltip on non-label part of edge"},
		{Name: "edgeURL", UsedBy: Edge, Documentation: "Link for non-label parts of edge"},
		{Name: "epsilon", UsedBy: Graph, Documentation: "Terminating condition (neato layout)"},
		{Name: "esep", UsedBy: Graph, Documentation: "Margin around polygons for spline edge routing"},
		{Name: "fillcolor", UsedBy: Node | Edge | Cluster, Documentation: "Color for filling node or cluster background"},
		{Name: "fixedsize", UsedBy: Node, Documentation: "Use specified width/height for node size"},
		{Name: "fontcolor", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Color used for text"},
		{Name: "fontname", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Font used for text"},
		{Name: "fontnames", UsedBy: Graph, Documentation: "Control fontname representation in SVG"},
		{Name: "fontpath", UsedBy: Graph, Documentation: "Directory list for bitmap font search"},
		{Name: "fontsize", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Font size in points"},
		{Name: "forcelabels", UsedBy: Graph, Documentation: "Force placement of all xlabels"},
		{Name: "gradientangle", UsedBy: Node | Cluster | Graph, Documentation: "Angle of gradient fill"},
		{Name: "group", UsedBy: Node, Documentation: "Name for node group with bundled edges (dot)"},
		{Name: "head_lp", UsedBy: Edge, Documentation: "Center position of edge head label (write-only)"},
		{Name: "headclip", UsedBy: Edge, Documentation: "Clip edge head to node boundary"},
		{Name: "headhref", UsedBy: Edge, Documentation: "Synonym for headURL"},
		{Name: "headlabel", UsedBy: Edge, Documentation: "Text label near head of edge"},
		{Name: "headport", UsedBy: Edge, Documentation: "Where on head node to attach edge"},
		{Name: "headtarget", UsedBy: Edge, Documentation: "Browser window for headURL link"},
		{Name: "headtooltip", UsedBy: Edge, Documentation: "Tooltip on edge head"},
		{Name: "headURL", UsedBy: Edge, Documentation: "Link for edge head label"},
		{Name: "height", UsedBy: Node, Documentation: "Height of node in inches"},
		{Name: "href", UsedBy: Graph | Cluster | Node | Edge, Documentation: "Synonym for URL"},
		{Name: "id", UsedBy: Graph | Cluster | Node | Edge, Documentation: "Identifier for graph objects"},
		{Name: "image", UsedBy: Node, Documentation: "File containing image for node"},
		{Name: "imagepath", UsedBy: Graph, Documentation: "Directories to search for image files"},
		{Name: "imagepos", UsedBy: Node, Documentation: "Position of image within node"},
		{Name: "imagescale", UsedBy: Node, Documentation: "How image fills containing node"},
		{Name: "inputscale", UsedBy: Graph, Documentation: "Scales input positions to convert length units"},
		{Name: "K", UsedBy: Graph | Cluster, Documentation: "Spring constant for virtual physical model"},
		{Name: "label", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Text label attached to objects"},
		{Name: "label_scheme", UsedBy: Graph, Documentation: "Treat special nodes as edge labels (sfdp)"},
		{Name: "labelangle", UsedBy: Edge, Documentation: "Angle in degrees of head/tail edge labels"},
		{Name: "labeldistance", UsedBy: Edge, Documentation: "Scaling factor for head/tail label distance"},
		{Name: "labelfloat", UsedBy: Edge, Documentation: "Allow edge labels less constrained in position"},
		{Name: "labelfontcolor", UsedBy: Edge, Documentation: "Color for headlabel and taillabel"},
		{Name: "labelfontname", UsedBy: Edge, Documentation: "Font for headlabel and taillabel"},
		{Name: "labelfontsize", UsedBy: Edge, Documentation: "Font size for headlabel and taillabel"},
		{Name: "labelhref", UsedBy: Edge, Documentation: "Synonym for labelURL"},
		{Name: "labeljust", UsedBy: Graph | Cluster, Documentation: "Justification for graph/cluster labels"},
		{Name: "labelloc", UsedBy: Node | Graph | Cluster, Documentation: "Vertical placement of labels"},
		{Name: "labeltarget", UsedBy: Edge, Documentation: "Browser window for labelURL links"},
		{Name: "labeltooltip", UsedBy: Edge, Documentation: "Tooltip on edge label"},
		{Name: "labelURL", UsedBy: Edge, Documentation: "Link for edge label"},
		{Name: "landscape", UsedBy: Graph, Documentation: "Render graph in landscape mode"},
		{Name: "layer", UsedBy: Edge | Node | Cluster, Documentation: "Specifies layers for object presence"},
		{Name: "layerlistsep", UsedBy: Graph, Documentation: "Separator for layerRange splitting"},
		{Name: "layers", UsedBy: Graph, Documentation: "Linearly ordered list of layer names"},
		{Name: "layerselect", UsedBy: Graph, Documentation: "Selects layers to be emitted"},
		{Name: "layersep", UsedBy: Graph, Documentation: "Separator for layers attribute splitting"},
		{Name: "layout", Type: TypeLayout, UsedBy: Graph, Documentation: "Which layout engine to use"},
		{Name: "len", UsedBy: Edge, Documentation: "Preferred edge length in inches"},
		{Name: "levels", UsedBy: Graph, Documentation: "Levels allowed in multilevel scheme (sfdp)"},
		{Name: "levelsgap", UsedBy: Graph, Documentation: "Strictness of neato level constraints"},
		{Name: "lhead", UsedBy: Edge, Documentation: "Logical head of edge (dot layout)"},
		{Name: "lheight", UsedBy: Graph | Cluster, Documentation: "Height of graph/cluster label (write-only)"},
		{Name: "linelength", UsedBy: Graph, Documentation: "String length before overflow to next line"},
		{Name: "lp", UsedBy: Edge | Graph | Cluster, Documentation: "Label center position (write-only)"},
		{Name: "ltail", UsedBy: Edge, Documentation: "Logical tail of edge (dot layout)"},
		{Name: "lwidth", UsedBy: Graph | Cluster, Documentation: "Width of graph/cluster label (write-only)"},
		{Name: "margin", UsedBy: Node | Cluster | Graph, Documentation: "Margin around canvas or node content"},
		{Name: "maxiter", UsedBy: Graph, Documentation: "Number of iterations for layout"},
		{Name: "mclimit", UsedBy: Graph, Documentation: "Scale factor for mincross edge crossing minimizer"},
		{Name: "mindist", UsedBy: Graph, Documentation: "Minimum separation between all nodes (circo)"},
		{Name: "minlen", UsedBy: Edge, Documentation: "Minimum edge length by rank difference (dot)"},
		{Name: "mode", UsedBy: Graph, Documentation: "Technique for layout optimization (neato)"},
		{Name: "model", UsedBy: Graph, Documentation: "Distance matrix computation method (neato)"},
		{Name: "newrank", UsedBy: Graph, Documentation: "Use single global ranking, ignoring clusters (dot)"},
		{Name: "nodesep", UsedBy: Graph, Documentation: "Minimum space between adjacent nodes"},
		{Name: "nojustify", UsedBy: Graph | Cluster | Node | Edge, Documentation: "Justify multiline text vs previous line"},
		{Name: "normalize", UsedBy: Graph, Documentation: "Normalize final layout coordinates"},
		{Name: "notranslate", UsedBy: Graph, Documentation: "Avoid translating layout to origin (neato)"},
		{Name: "nslimit", UsedBy: Graph, Documentation: "Iterations in network simplex (dot)"},
		{Name: "nslimit1", UsedBy: Graph, Documentation: "Iterations in network simplex for ranking (dot)"},
		{Name: "oneblock", UsedBy: Graph, Documentation: "Draw circo graphs around one circle"},
		{Name: "ordering", UsedBy: Graph | Node, Documentation: "Constrain left-to-right edge ordering (dot)"},
		{Name: "orientation", UsedBy: Node | Graph, Documentation: "Node rotation angle or graph orientation"},
		{Name: "outputorder", UsedBy: Graph, Documentation: "Order for drawing nodes and edges"},
		{Name: "overlap", UsedBy: Graph, Documentation: "Remove or determine node overlaps"},
		{Name: "overlap_scaling", UsedBy: Graph, Documentation: "Scale layout to reduce node overlap"},
		{Name: "overlap_shrink", UsedBy: Graph, Documentation: "Compress pass for overlap removal"},
		{Name: "pack", UsedBy: Graph, Documentation: "Layout components separately then pack"},
		{Name: "packmode", UsedBy: Graph, Documentation: "How connected components should be packed"},
		{Name: "pad", UsedBy: Graph, Documentation: "Inches extending drawing area around graph"},
		{Name: "page", UsedBy: Graph, Documentation: "Width and height of output pages"},
		{Name: "pagedir", UsedBy: Graph, Documentation: "Order in which pages are emitted"},
		{Name: "pencolor", UsedBy: Cluster, Documentation: "Color for cluster bounding box"},
		{Name: "penwidth", UsedBy: Cluster | Node | Edge, Documentation: "Width of pen for drawing lines/curves"},
		{Name: "peripheries", UsedBy: Node | Cluster, Documentation: "Number of peripheries in shapes/boundaries"},
		{Name: "pin", UsedBy: Node, Documentation: "Keep node at input position (neato, fdp)"},
		{Name: "pos", UsedBy: Edge | Node, Documentation: "Position of node or spline control points"},
		{Name: "quadtree", UsedBy: Graph, Documentation: "Quadtree scheme for layout (sfdp)"},
		{Name: "quantum", UsedBy: Graph, Documentation: "Round node label dimensions to quantum multiples"},
		{Name: "radius", UsedBy: Edge, Documentation: "Radius of rounded corners on orthogonal edges"},
		{Name: "rank", UsedBy: Subgraph, Documentation: "Rank constraints on subgraph nodes (dot)"},
		{Name: "rankdir", UsedBy: Graph, Documentation: "Sets direction of graph layout (dot)"},
		{Name: "ranksep", UsedBy: Graph, Documentation: "Specifies separation between ranks"},
		{Name: "ratio", UsedBy: Graph, Documentation: "Aspect ratio for drawing"},
		{Name: "rects", UsedBy: Node, Documentation: "Rectangles for record fields (write-only)"},
		{Name: "regular", UsedBy: Node, Documentation: "Force polygon to be regular"},
		{Name: "remincross", UsedBy: Graph, Documentation: "Run edge crossing minimization twice (dot)"},
		{Name: "repulsiveforce", UsedBy: Graph, Documentation: "Power of repulsive force (sfdp)"},
		{Name: "resolution", UsedBy: Graph, Documentation: "Synonym for dpi"},
		{Name: "root", UsedBy: Graph | Node, Documentation: "Nodes for layout center (twopi, circo)"},
		{Name: "rotate", UsedBy: Graph, Documentation: "Sets drawing orientation to landscape"},
		{Name: "rotation", UsedBy: Graph, Documentation: "Rotate final layout counter-clockwise (sfdp)"},
		{Name: "samehead", UsedBy: Edge, Documentation: "Aim edges at same head point (dot)"},
		{Name: "sametail", UsedBy: Edge, Documentation: "Aim edges at same tail point (dot)"},
		{Name: "samplepoints", UsedBy: Node, Documentation: "Points used for circle/ellipse node"},
		{Name: "scale", UsedBy: Graph, Documentation: "Scale layout by factor after initial layout"},
		{Name: "searchsize", UsedBy: Graph, Documentation: "Max edges to search for minimum cut (dot)"},
		{Name: "sep", UsedBy: Graph, Documentation: "Margin around nodes when removing overlap"},
		{Name: "shape", UsedBy: Node, Documentation: "Shape of a node"},
		{Name: "shapefile", UsedBy: Node, Documentation: "File with user-supplied node content"},
		{Name: "showboxes", UsedBy: Edge | Node | Graph, Documentation: "Print guide boxes for debugging (dot)"},
		{Name: "sides", UsedBy: Node, Documentation: "Number of sides for polygon shape"},
		{Name: "size", UsedBy: Graph, Documentation: "Maximum width and height of drawing"},
		{Name: "skew", UsedBy: Node, Documentation: "Skew factor for polygon shapes"},
		{Name: "smoothing", UsedBy: Graph, Documentation: "Post-processing step for node distribution (sfdp)"},
		{Name: "sortv", UsedBy: Graph | Cluster | Node, Documentation: "Sort order for component packing"},
		{Name: "splines", UsedBy: Graph, Documentation: "How edges are represented"},
		{Name: "start", UsedBy: Graph, Documentation: "Parameter for initial node layout"},
		{Name: "style", UsedBy: Edge | Node | Cluster | Graph, Documentation: "Style information for graph components"},
		{Name: "stylesheet", UsedBy: Graph, Documentation: "XML style sheet for SVG output"},
		{Name: "tail_lp", UsedBy: Edge, Documentation: "Position of edge tail label (write-only)"},
		{Name: "tailclip", UsedBy: Edge, Documentation: "Clip edge tail to node boundary"},
		{Name: "tailhref", UsedBy: Edge, Documentation: "Synonym for tailURL"},
		{Name: "taillabel", UsedBy: Edge, Documentation: "Text label near tail of edge"},
		{Name: "tailport", UsedBy: Edge, Documentation: "Where on tail node to attach edge"},
		{Name: "tailtarget", UsedBy: Edge, Documentation: "Browser window for tailURL link"},
		{Name: "tailtooltip", UsedBy: Edge, Documentation: "Tooltip on edge tail"},
		{Name: "tailURL", UsedBy: Edge, Documentation: "Link for edge tail label"},
		{Name: "target", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Browser window for object URL"},
		{Name: "TBbalance", UsedBy: Graph, Documentation: "Move floating nodes to min/max rank (dot)"},
		{Name: "tooltip", UsedBy: Node | Edge | Cluster | Graph, Documentation: "Tooltip text on hover"},
		{Name: "truecolor", UsedBy: Graph, Documentation: "Use truecolor or palette for bitmap rendering"},
		{Name: "URL", UsedBy: Edge | Node | Graph | Cluster, Documentation: "Hyperlinks in device-dependent output"},
		{Name: "vertices", UsedBy: Node, Documentation: "Polygon vertex coordinates (write-only)"},
		{Name: "viewport", UsedBy: Graph, Documentation: "Clipping window on final drawing"},
		{Name: "voro_margin", UsedBy: Graph, Documentation: "Tuning margin for Voronoi technique"},
		{Name: "weight", UsedBy: Edge, Documentation: "Weight of edge"},
		{Name: "width", UsedBy: Node, Documentation: "Width of node in inches"},
		{Name: "xdotversion", UsedBy: Graph, Documentation: "Version of xdot used in output"},
		{Name: "xlabel", UsedBy: Edge | Node, Documentation: "External label for node or edge"},
		{Name: "xlp", UsedBy: Node | Edge, Documentation: "Position of exterior label (write-only)"},
		{Name: "z", UsedBy: Node, Documentation: "Z-coordinate for 3D layouts"},
	}
	slices.SortFunc(attributes, func(a, b Attribute) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range attributes {
		attributes[i].MarkdownDoc = attributes[i].markdownDoc()
	}

	return attributes
}()
