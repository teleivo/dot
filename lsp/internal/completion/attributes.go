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

// AttrValue represents a valid value for an attribute type with its applicable contexts.
type AttrValue struct {
	Value  string           // The value string (e.g., "dashed", "filled")
	UsedBy AttributeContext // Which contexts this value is valid for (0 means all)
}

const (
	TypeUnknown AttrType = iota
	TypeArrowType
	TypeBool
	TypeDirType
	TypeLayout
	TypeStyle
)

// attrTypeInfo holds metadata for each AttrType, indexed by the type value.
var attrTypeInfo = [...]struct {
	// Name is the type name as used in Graphviz documentation (e.g., "dirType").
	Name string
	// Values contains valid values for this type. May not be exhaustive for
	// complex types like arrowType where values can be combined.
	Values []AttrValue
	// Doc is a brief description of what the type represents.
	Doc string
}{
	TypeUnknown:   {"", nil, ""},
	TypeArrowType: {"arrowType", av("box", "crow", "curve", "diamond", "dot", "icurve", "inv", "none", "normal", "tee", "vee"), "Arrow shape"},
	TypeBool:      {"bool", av("true", "false", "yes", "no"), "Boolean value"},
	TypeDirType:   {"dirType", av("back", "both", "forward", "none"), "Edge arrow direction"},
	TypeLayout:    {"layout", av("circo", "dot", "fdp", "neato", "osage", "patchwork", "sfdp", "twopi"), "Layout engine name"},
	TypeStyle: {"style", []AttrValue{
		{Value: "solid", UsedBy: Node | Edge},
		{Value: "dashed", UsedBy: Node | Edge},
		{Value: "dotted", UsedBy: Node | Edge},
		{Value: "bold", UsedBy: Node | Edge},
		{Value: "invis", UsedBy: Node | Edge},
		{Value: "filled", UsedBy: Node | Edge | Cluster},
		{Value: "striped", UsedBy: Node | Cluster},
		{Value: "wedged", UsedBy: Node},
		{Value: "diagonals", UsedBy: Node},
		{Value: "rounded", UsedBy: Node | Cluster},
		{Value: "tapered", UsedBy: Edge},
		{Value: "radial", UsedBy: Node | Cluster | Graph},
	}, "Drawing style"},
}

// av is a helper to create []AttrValue from strings where UsedBy is All.
func av(values ...string) []AttrValue {
	result := make([]AttrValue, len(values))
	for i, v := range values {
		result[i] = AttrValue{Value: v, UsedBy: All}
	}
	return result
}

// String returns the type name (e.g., "dirType").
func (t AttrType) String() string { return attrTypeInfo[t].Name }

// Values returns all valid values for this type (for documentation display).
func (t AttrType) Values() []AttrValue { return attrTypeInfo[t].Values }

// ValuesFor returns valid values filtered by context.
func (t AttrType) ValuesFor(attrCtx AttributeContext) []AttrValue {
	all := attrTypeInfo[t].Values
	var result []AttrValue
	for _, v := range all {
		if v.UsedBy&attrCtx != 0 {
			result = append(result, v)
		}
	}
	return result
}

// Doc returns a brief description of the type.
func (t AttrType) Doc() string { return attrTypeInfo[t].Doc }

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

	All = Graph | Subgraph | Cluster | Node | Edge // All contexts
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
	// Doc is a brief description of what the attribute does.
	Doc string
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
	sb.WriteString(a.Doc)
	sb.WriteString("\n\n")

	if a.Type != TypeUnknown {
		sb.WriteString("**Type:** [")
		sb.WriteString(a.Type.String())
		sb.WriteString("](")
		sb.WriteString(a.Type.URL())
		sb.WriteString(")")

		if values := a.Type.Values(); len(values) > 0 {
			sb.WriteString(": `")
			sb.WriteString(values[0].Value)
			for _, v := range values[1:] {
				sb.WriteString("` | `")
				sb.WriteString(v.Value)
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
		{Name: "_background", UsedBy: Graph, Doc: "Specifies arbitrary background using xdot format strings"},
		{Name: "area", UsedBy: Node | Cluster, Doc: "referred area for node or empty cluster (patchwork layout)"},
		{Name: "arrowhead", Type: TypeArrowType, UsedBy: Edge, Doc: "Style of arrowhead on edge head node"},
		{Name: "arrowsize", UsedBy: Edge, Doc: "Multiplicative scale factor for arrowheads"},
		{Name: "arrowtail", Type: TypeArrowType, UsedBy: Edge, Doc: "Style of arrowhead on edge tail node"},
		{Name: "bb", UsedBy: Cluster | Graph, Doc: "Bounding box of drawing in points (write-only)"},
		{Name: "beautify", Type: TypeBool, UsedBy: Graph, Doc: "Whether to draw leaf nodes in circle around root (sfdp)"},
		{Name: "bgcolor", UsedBy: Graph | Cluster, Doc: "Canvas background color"},
		{Name: "center", Type: TypeBool, UsedBy: Graph, Doc: "Whether to center drawing in output canvas"},
		{Name: "charset", UsedBy: Graph, Doc: "Character encoding for text labels"},
		{Name: "class", UsedBy: Edge | Node | Cluster | Graph, Doc: "Classnames for SVG element styling"},
		{Name: "cluster", Type: TypeBool, UsedBy: Cluster | Subgraph, Doc: "Whether subgraph is a cluster"},
		{Name: "clusterrank", UsedBy: Graph, Doc: "Mode for handling clusters (dot layout)"},
		{Name: "color", UsedBy: Edge | Node | Cluster, Doc: "Basic drawing color for graphics"},
		{Name: "colorscheme", UsedBy: Edge | Node | Cluster | Graph, Doc: "Color scheme namespace for interpreting color names"},
		{Name: "comment", UsedBy: Edge | Node | Graph, Doc: "Comments inserted into output"},
		{Name: "compound", Type: TypeBool, UsedBy: Graph, Doc: "Allow edges between clusters (dot layout)"},
		{Name: "concentrate", Type: TypeBool, UsedBy: Graph, Doc: "Use edge concentrators"},
		{Name: "constraint", Type: TypeBool, UsedBy: Edge, Doc: "Whether edge used in node ranking (dot layout)"},
		{Name: "Damping", UsedBy: Graph, Doc: "Factor damping force motions (neato layout)"},
		{Name: "decorate", Type: TypeBool, UsedBy: Edge, Doc: "Connect edge label to edge with line"},
		{Name: "defaultdist", UsedBy: Graph, Doc: "Distance between nodes in separate components (neato)"},
		{Name: "dim", UsedBy: Graph, Doc: "Number of dimensions for layout"},
		{Name: "dimen", UsedBy: Graph, Doc: "Number of dimensions for rendering"},
		{Name: "dir", Type: TypeDirType, UsedBy: Edge, Doc: "Edge type for drawing arrowheads"},
		{Name: "diredgeconstraints", UsedBy: Graph, Doc: "Constrain edges to point downwards (neato)"},
		{Name: "distortion", UsedBy: Node, Doc: "Distortion factor for polygon shapes"},
		{Name: "dpi", UsedBy: Graph, Doc: "Expected pixels per inch on display device"},
		{Name: "edgehref", UsedBy: Edge, Doc: "Synonym for edgeURL"},
		{Name: "edgetarget", UsedBy: Edge, Doc: "Browser window for edgeURL link"},
		{Name: "edgetooltip", UsedBy: Edge, Doc: "Tooltip on non-label part of edge"},
		{Name: "edgeURL", UsedBy: Edge, Doc: "Link for non-label parts of edge"},
		{Name: "epsilon", UsedBy: Graph, Doc: "Terminating condition (neato layout)"},
		{Name: "esep", UsedBy: Graph, Doc: "Margin around polygons for spline edge routing"},
		{Name: "fillcolor", UsedBy: Node | Edge | Cluster, Doc: "Color for filling node or cluster background"},
		{Name: "fixedsize", Type: TypeBool, UsedBy: Node, Doc: "Use specified width/height for node size"},
		{Name: "fontcolor", UsedBy: Edge | Node | Graph | Cluster, Doc: "Color used for text"},
		{Name: "fontname", UsedBy: Edge | Node | Graph | Cluster, Doc: "Font used for text"},
		{Name: "fontnames", UsedBy: Graph, Doc: "Control fontname representation in SVG"},
		{Name: "fontpath", UsedBy: Graph, Doc: "Directory list for bitmap font search"},
		{Name: "fontsize", UsedBy: Edge | Node | Graph | Cluster, Doc: "Font size in points"},
		{Name: "forcelabels", Type: TypeBool, UsedBy: Graph, Doc: "Force placement of all xlabels"},
		{Name: "gradientangle", UsedBy: Node | Cluster | Graph, Doc: "Angle of gradient fill"},
		{Name: "group", UsedBy: Node, Doc: "Name for node group with bundled edges (dot)"},
		{Name: "head_lp", UsedBy: Edge, Doc: "Center position of edge head label (write-only)"},
		{Name: "headclip", Type: TypeBool, UsedBy: Edge, Doc: "Clip edge head to node boundary"},
		{Name: "headhref", UsedBy: Edge, Doc: "Synonym for headURL"},
		{Name: "headlabel", UsedBy: Edge, Doc: "Text label near head of edge"},
		{Name: "headport", UsedBy: Edge, Doc: "Where on head node to attach edge"},
		{Name: "headtarget", UsedBy: Edge, Doc: "Browser window for headURL link"},
		{Name: "headtooltip", UsedBy: Edge, Doc: "Tooltip on edge head"},
		{Name: "headURL", UsedBy: Edge, Doc: "Link for edge head label"},
		{Name: "height", UsedBy: Node, Doc: "Height of node in inches"},
		{Name: "href", UsedBy: Graph | Cluster | Node | Edge, Doc: "Synonym for URL"},
		{Name: "id", UsedBy: Graph | Cluster | Node | Edge, Doc: "Identifier for graph objects"},
		{Name: "image", UsedBy: Node, Doc: "File containing image for node"},
		{Name: "imagepath", UsedBy: Graph, Doc: "Directories to search for image files"},
		{Name: "imagepos", UsedBy: Node, Doc: "Position of image within node"},
		{Name: "imagescale", UsedBy: Node, Doc: "How image fills containing node"},
		{Name: "inputscale", UsedBy: Graph, Doc: "Scales input positions to convert length units"},
		{Name: "K", UsedBy: Graph | Cluster, Doc: "Spring constant for virtual physical model"},
		{Name: "label", UsedBy: Edge | Node | Graph | Cluster, Doc: "Text label attached to objects"},
		{Name: "label_scheme", UsedBy: Graph, Doc: "Treat special nodes as edge labels (sfdp)"},
		{Name: "labelangle", UsedBy: Edge, Doc: "Angle in degrees of head/tail edge labels"},
		{Name: "labeldistance", UsedBy: Edge, Doc: "Scaling factor for head/tail label distance"},
		{Name: "labelfloat", Type: TypeBool, UsedBy: Edge, Doc: "Allow edge labels less constrained in position"},
		{Name: "labelfontcolor", UsedBy: Edge, Doc: "Color for headlabel and taillabel"},
		{Name: "labelfontname", UsedBy: Edge, Doc: "Font for headlabel and taillabel"},
		{Name: "labelfontsize", UsedBy: Edge, Doc: "Font size for headlabel and taillabel"},
		{Name: "labelhref", UsedBy: Edge, Doc: "Synonym for labelURL"},
		{Name: "labeljust", UsedBy: Graph | Cluster, Doc: "Justification for graph/cluster labels"},
		{Name: "labelloc", UsedBy: Node | Graph | Cluster, Doc: "Vertical placement of labels"},
		{Name: "labeltarget", UsedBy: Edge, Doc: "Browser window for labelURL links"},
		{Name: "labeltooltip", UsedBy: Edge, Doc: "Tooltip on edge label"},
		{Name: "labelURL", UsedBy: Edge, Doc: "Link for edge label"},
		{Name: "landscape", Type: TypeBool, UsedBy: Graph, Doc: "Render graph in landscape mode"},
		{Name: "layer", UsedBy: Edge | Node | Cluster, Doc: "Specifies layers for object presence"},
		{Name: "layerlistsep", UsedBy: Graph, Doc: "Separator for layerRange splitting"},
		{Name: "layers", UsedBy: Graph, Doc: "Linearly ordered list of layer names"},
		{Name: "layerselect", UsedBy: Graph, Doc: "Selects layers to be emitted"},
		{Name: "layersep", UsedBy: Graph, Doc: "Separator for layers attribute splitting"},
		{Name: "layout", Type: TypeLayout, UsedBy: Graph, Doc: "Which layout engine to use"},
		{Name: "len", UsedBy: Edge, Doc: "Preferred edge length in inches"},
		{Name: "levels", UsedBy: Graph, Doc: "Levels allowed in multilevel scheme (sfdp)"},
		{Name: "levelsgap", UsedBy: Graph, Doc: "Strictness of neato level constraints"},
		{Name: "lhead", UsedBy: Edge, Doc: "Logical head of edge (dot layout)"},
		{Name: "lheight", UsedBy: Graph | Cluster, Doc: "Height of graph/cluster label (write-only)"},
		{Name: "linelength", UsedBy: Graph, Doc: "String length before overflow to next line"},
		{Name: "lp", UsedBy: Edge | Graph | Cluster, Doc: "Label center position (write-only)"},
		{Name: "ltail", UsedBy: Edge, Doc: "Logical tail of edge (dot layout)"},
		{Name: "lwidth", UsedBy: Graph | Cluster, Doc: "Width of graph/cluster label (write-only)"},
		{Name: "margin", UsedBy: Node | Cluster | Graph, Doc: "Margin around canvas or node content"},
		{Name: "maxiter", UsedBy: Graph, Doc: "Number of iterations for layout"},
		{Name: "mclimit", UsedBy: Graph, Doc: "Scale factor for mincross edge crossing minimizer"},
		{Name: "mindist", UsedBy: Graph, Doc: "Minimum separation between all nodes (circo)"},
		{Name: "minlen", UsedBy: Edge, Doc: "Minimum edge length by rank difference (dot)"},
		{Name: "mode", UsedBy: Graph, Doc: "Technique for layout optimization (neato)"},
		{Name: "model", UsedBy: Graph, Doc: "Distance matrix computation method (neato)"},
		{Name: "newrank", Type: TypeBool, UsedBy: Graph, Doc: "Use single global ranking, ignoring clusters (dot)"},
		{Name: "nodesep", UsedBy: Graph, Doc: "Minimum space between adjacent nodes"},
		{Name: "nojustify", Type: TypeBool, UsedBy: Graph | Cluster | Node | Edge, Doc: "Justify multiline text vs previous line"},
		{Name: "normalize", UsedBy: Graph, Doc: "Normalize final layout coordinates"},
		{Name: "notranslate", Type: TypeBool, UsedBy: Graph, Doc: "Avoid translating layout to origin (neato)"},
		{Name: "nslimit", UsedBy: Graph, Doc: "Iterations in network simplex (dot)"},
		{Name: "nslimit1", UsedBy: Graph, Doc: "Iterations in network simplex for ranking (dot)"},
		{Name: "oneblock", Type: TypeBool, UsedBy: Graph, Doc: "Draw circo graphs around one circle"},
		{Name: "ordering", UsedBy: Graph | Node, Doc: "Constrain left-to-right edge ordering (dot)"},
		{Name: "orientation", UsedBy: Node | Graph, Doc: "Node rotation angle or graph orientation"},
		{Name: "outputorder", UsedBy: Graph, Doc: "Order for drawing nodes and edges"},
		{Name: "overlap", UsedBy: Graph, Doc: "Remove or determine node overlaps"},
		{Name: "overlap_scaling", UsedBy: Graph, Doc: "Scale layout to reduce node overlap"},
		{Name: "overlap_shrink", Type: TypeBool, UsedBy: Graph, Doc: "Compress pass for overlap removal"},
		{Name: "pack", Type: TypeBool, UsedBy: Graph, Doc: "Layout components separately then pack"},
		{Name: "packmode", UsedBy: Graph, Doc: "How connected components should be packed"},
		{Name: "pad", UsedBy: Graph, Doc: "Inches extending drawing area around graph"},
		{Name: "page", UsedBy: Graph, Doc: "Width and height of output pages"},
		{Name: "pagedir", UsedBy: Graph, Doc: "Order in which pages are emitted"},
		{Name: "pencolor", UsedBy: Cluster, Doc: "Color for cluster bounding box"},
		{Name: "penwidth", UsedBy: Cluster | Node | Edge, Doc: "Width of pen for drawing lines/curves"},
		{Name: "peripheries", UsedBy: Node | Cluster, Doc: "Number of peripheries in shapes/boundaries"},
		{Name: "pin", Type: TypeBool, UsedBy: Node, Doc: "Keep node at input position (neato, fdp)"},
		{Name: "pos", UsedBy: Edge | Node, Doc: "Position of node or spline control points"},
		{Name: "quadtree", UsedBy: Graph, Doc: "Quadtree scheme for layout (sfdp)"},
		{Name: "quantum", UsedBy: Graph, Doc: "Round node label dimensions to quantum multiples"},
		{Name: "radius", UsedBy: Edge, Doc: "Radius of rounded corners on orthogonal edges"},
		{Name: "rank", UsedBy: Subgraph, Doc: "Rank constraints on subgraph nodes (dot)"},
		{Name: "rankdir", UsedBy: Graph, Doc: "Sets direction of graph layout (dot)"},
		{Name: "ranksep", UsedBy: Graph, Doc: "Specifies separation between ranks"},
		{Name: "ratio", UsedBy: Graph, Doc: "Aspect ratio for drawing"},
		{Name: "rects", UsedBy: Node, Doc: "Rectangles for record fields (write-only)"},
		{Name: "regular", Type: TypeBool, UsedBy: Node, Doc: "Force polygon to be regular"},
		{Name: "remincross", Type: TypeBool, UsedBy: Graph, Doc: "Run edge crossing minimization twice (dot)"},
		{Name: "repulsiveforce", UsedBy: Graph, Doc: "Power of repulsive force (sfdp)"},
		{Name: "resolution", UsedBy: Graph, Doc: "Synonym for dpi"},
		{Name: "root", UsedBy: Graph | Node, Doc: "Nodes for layout center (twopi, circo)"},
		{Name: "rotate", UsedBy: Graph, Doc: "Sets drawing orientation to landscape"},
		{Name: "rotation", UsedBy: Graph, Doc: "Rotate final layout counter-clockwise (sfdp)"},
		{Name: "samehead", UsedBy: Edge, Doc: "Aim edges at same head point (dot)"},
		{Name: "sametail", UsedBy: Edge, Doc: "Aim edges at same tail point (dot)"},
		{Name: "samplepoints", UsedBy: Node, Doc: "Points used for circle/ellipse node"},
		{Name: "scale", UsedBy: Graph, Doc: "Scale layout by factor after initial layout"},
		{Name: "searchsize", UsedBy: Graph, Doc: "Max edges to search for minimum cut (dot)"},
		{Name: "sep", UsedBy: Graph, Doc: "Margin around nodes when removing overlap"},
		{Name: "shape", UsedBy: Node, Doc: "Shape of a node"},
		{Name: "shapefile", UsedBy: Node, Doc: "File with user-supplied node content"},
		{Name: "showboxes", UsedBy: Edge | Node | Graph, Doc: "Print guide boxes for debugging (dot)"},
		{Name: "sides", UsedBy: Node, Doc: "Number of sides for polygon shape"},
		{Name: "size", UsedBy: Graph, Doc: "Maximum width and height of drawing"},
		{Name: "skew", UsedBy: Node, Doc: "Skew factor for polygon shapes"},
		{Name: "smoothing", UsedBy: Graph, Doc: "Post-processing step for node distribution (sfdp)"},
		{Name: "sortv", UsedBy: Graph | Cluster | Node, Doc: "Sort order for component packing"},
		{Name: "splines", UsedBy: Graph, Doc: "How edges are represented"},
		{Name: "start", UsedBy: Graph, Doc: "Parameter for initial node layout"},
		{Name: "style", Type: TypeStyle, UsedBy: Edge | Node | Cluster | Graph, Doc: "Style information for graph components"},
		{Name: "stylesheet", UsedBy: Graph, Doc: "XML style sheet for SVG output"},
		{Name: "tail_lp", UsedBy: Edge, Doc: "Position of edge tail label (write-only)"},
		{Name: "tailclip", Type: TypeBool, UsedBy: Edge, Doc: "Clip edge tail to node boundary"},
		{Name: "tailhref", UsedBy: Edge, Doc: "Synonym for tailURL"},
		{Name: "taillabel", UsedBy: Edge, Doc: "Text label near tail of edge"},
		{Name: "tailport", UsedBy: Edge, Doc: "Where on tail node to attach edge"},
		{Name: "tailtarget", UsedBy: Edge, Doc: "Browser window for tailURL link"},
		{Name: "tailtooltip", UsedBy: Edge, Doc: "Tooltip on edge tail"},
		{Name: "tailURL", UsedBy: Edge, Doc: "Link for edge tail label"},
		{Name: "target", UsedBy: Edge | Node | Graph | Cluster, Doc: "Browser window for object URL"},
		{Name: "TBbalance", UsedBy: Graph, Doc: "Move floating nodes to min/max rank (dot)"},
		{Name: "tooltip", UsedBy: Node | Edge | Cluster | Graph, Doc: "Tooltip text on hover"},
		{Name: "truecolor", Type: TypeBool, UsedBy: Graph, Doc: "Use truecolor or palette for bitmap rendering"},
		{Name: "URL", UsedBy: Edge | Node | Graph | Cluster, Doc: "Hyperlinks in device-dependent output"},
		{Name: "vertices", UsedBy: Node, Doc: "Polygon vertex coordinates (write-only)"},
		{Name: "viewport", UsedBy: Graph, Doc: "Clipping window on final drawing"},
		{Name: "voro_margin", UsedBy: Graph, Doc: "Tuning margin for Voronoi technique"},
		{Name: "weight", UsedBy: Edge, Doc: "Weight of edge"},
		{Name: "width", UsedBy: Node, Doc: "Width of node in inches"},
		{Name: "xdotversion", UsedBy: Graph, Doc: "Version of xdot used in output"},
		{Name: "xlabel", UsedBy: Edge | Node, Doc: "External label for node or edge"},
		{Name: "xlp", UsedBy: Node | Edge, Doc: "Position of exterior label (write-only)"},
		{Name: "z", UsedBy: Node, Doc: "Z-coordinate for 3D layouts"},
	}
	slices.SortFunc(attributes, func(a, b Attribute) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range attributes {
		attributes[i].MarkdownDoc = attributes[i].markdownDoc()
	}

	return attributes
}()
