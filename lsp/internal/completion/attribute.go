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
	UsedBy AttributeContext // Which contexts this value is valid for
	Doc    string           // Brief description of what the value does
}

// markdownDoc generates the markdown documentation for this value.
func (v AttrValue) markdownDoc(attrType AttrType) string {
	var sb strings.Builder
	if v.Doc != "" {
		sb.WriteString(v.Doc)
		sb.WriteString("\n\n")
	}
	sb.WriteString("[")
	sb.WriteString(attrType.String())
	sb.WriteString("](")
	sb.WriteString(attrType.URL())
	sb.WriteString(")")
	return sb.String()
}

const (
	TypeUnknown AttrType = iota
	TypeAddDouble
	TypeAddPoint
	TypeArrowType
	TypeBool
	TypeClusterMode
	TypeColor
	TypeColorList
	TypeDirType
	TypeDouble
	TypeDoubleList
	TypeEscString
	TypeInt
	TypeLayerList
	TypeLayerRange
	TypeLblString
	TypeLayout
	TypeOutputMode
	TypePackMode
	TypePagedir
	TypePoint
	TypePointList
	TypePortPos
	TypeQuadType
	TypeRankdir
	TypeRankType
	TypeRect
	TypeShape
	TypeSmoothType
	TypeSplineType
	TypeStartType
	TypeString
	TypeStyle
	TypeViewPort
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
	TypeUnknown:     {"", nil, ""},
	TypeAddDouble:   {"addDouble", nil, "Double with optional + prefix to add to default. Format: [+]number"},
	TypeAddPoint:    {"addPoint", nil, "Point with optional + prefix for vector addition. Format: [+]x,y[,z][!]"},
	TypeArrowType:   {"arrowType", av("box", "crow", "curve", "diamond", "dot", "icurve", "inv", "none", "normal", "tee", "vee"), "Edge arrowhead shape"},
	TypeBool:        {"bool", av("false", "no", "true", "yes"), "Boolean value"},
	TypeClusterMode: {"clusterMode", av("global", "local", "none"), "Cluster handling mode"},
	TypeColor:       {"color", nil, "Color value. Format: #rrggbb, #rrggbbaa, H,S,V, or name"},
	TypeColorList:   {"colorList", nil, "Weighted color list for gradients. Format: color[:color]* or color;weight[:...]"},
	TypeDirType:     {"dirType", av("back", "both", "forward", "none"), "Edge arrow direction"},
	TypeDouble:      {"double", nil, "Double-precision floating point number"},
	TypeDoubleList:  {"doubleList", nil, "Colon-separated list of doubles. Format: num[:num]*"},
	TypeEscString:   {"escString", nil, "String with escape sequences. Escapes: \\N \\G \\E \\T \\H \\L \\n \\l \\r"},
	TypeInt:         {"int", nil, "Integer"},
	TypeLayerList:   {"layerList", nil, "List of layer names. Separator: layersep (default :)"},
	TypeLayerRange:  {"layerRange", nil, "Layer range specification. Format: layer or layer1:layer2"},
	TypeLblString:   {"lblString", nil, "Label: escString or HTML-like <table>...</table>"},
	TypeLayout: {"layout", []AttrValue{
		{Value: "circo", UsedBy: All, Doc: "Circular layout for cyclic structures"},
		{Value: "dot", UsedBy: All, Doc: "Hierarchical layout for directed graphs"},
		{Value: "fdp", UsedBy: All, Doc: "Force-directed layout using springs"},
		{Value: "neato", UsedBy: All, Doc: "Force-directed layout using stress majorization"},
		{Value: "osage", UsedBy: All, Doc: "Array-based layout for clustered graphs"},
		{Value: "patchwork", UsedBy: All, Doc: "Squarified treemap layout"},
		{Value: "sfdp", UsedBy: All, Doc: "Scalable force-directed layout for large graphs"},
		{Value: "twopi", UsedBy: All, Doc: "Radial layout with root at center"},
	}, "Layout engine"},
	TypeOutputMode: {"outputMode", av("breadthfirst", "edgesfirst", "nodesfirst"), "Order in which nodes and edges are drawn"},
	TypePackMode:   {"packMode", av("cluster", "graph", "node"), "How closely to pack graph components"},
	TypePagedir:    {"pagedir", av("BL", "BR", "LB", "LT", "RB", "RT", "TL", "TR"), "Page traversal order for multi-page output"},
	TypePoint:       {"point", nil, "2D/3D point. Format: x,y[,z][!] (! fixes position)"},
	TypePointList:   {"pointList", nil, "Space-separated list of points. Format: x,y x,y ..."},
	TypePortPos:     {"portPos", nil, "Port position on node. Format: portname[:compass]"},
	TypeQuadType: {"quadType", av("fast", "none", "normal"), "Quadtree scheme for force-directed layout"},
	TypeRankdir:  {"rankdir", av("BT", "LR", "RL", "TB"), "Graph layout direction"},
	TypeRankType: {"rankType", av("max", "min", "same", "sink", "source"), "Rank constraint on subgraph nodes"},
	TypeRect:        {"rect", nil, "Rectangle. Format: llx,lly,urx,ury"},
	TypeShape: {"shape", av(
		"Mcircle", "Mdiamond", "Mrecord", "Msquare",
		"assembly", "box", "box3d", "cds", "circle", "component", "cylinder",
		"diamond", "doublecircle", "doubleoctagon",
		"egg", "ellipse",
		"fivepoverhang", "folder",
		"hexagon", "house",
		"insulator", "invhouse", "invtrapezium", "invtriangle",
		"larrow", "lpromoter",
		"none", "note", "noverhang",
		"octagon", "oval",
		"parallelogram", "pentagon", "plain", "plaintext", "point", "polygon",
		"primersite", "promoter", "proteasesite", "proteinstab",
		"rarrow", "record", "rect", "rectangle", "restrictionsite", "ribosite",
		"rnastab", "rpromoter",
		"septagon", "signature", "square", "star",
		"tab", "terminator", "threepoverhang", "trapezium", "triangle", "tripleoctagon",
		"underline", "utr",
	), "Node shape"},
	TypeSmoothType: {"smoothType", av("avg_dist", "graph_dist", "none", "power_dist", "rng", "spring", "triangle"), "Post-processing smoothing for sfdp"},
	TypeSplineType:  {"splineType", nil, "Spline control points. Format: [e,x,y] [s,x,y] point (point point point)+"},
	TypeStartType:   {"startType", nil, "Initial node placement. Format: [style][seed]"},
	TypeString:      {"string", nil, "Text string"},
	TypeStyle: {"style", []AttrValue{
		{Value: "solid", UsedBy: Node | Edge},
		{Value: "dashed", UsedBy: Node | Edge},
		{Value: "dotted", UsedBy: Node | Edge},
		{Value: "bold", UsedBy: Node | Edge},
		{Value: "invis", UsedBy: Node | Edge},
		{Value: "filled", UsedBy: Node | Cluster},
		{Value: "striped", UsedBy: Node | Cluster},
		{Value: "wedged", UsedBy: Node},
		{Value: "diagonals", UsedBy: Node},
		{Value: "rounded", UsedBy: Node | Cluster},
		{Value: "tapered", UsedBy: Edge},
		{Value: "radial", UsedBy: Node | Cluster | Graph},
	}, "Drawing style"},
	TypeViewPort: {"viewPort", nil, "Clipping window. Format: W,H[,Z[,x,y]] or W,H,Z,'node'"},
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

// markdownDoc generates the markdown documentation for this type.
func (t AttrType) markdownDoc() string {
	if t == TypeUnknown {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(t.String())
	sb.WriteString("](")
	sb.WriteString(t.URL())
	sb.WriteString(")")

	if values := t.Values(); len(values) > 0 {
		sb.WriteString(": `")
		sb.WriteString(values[0].Value)
		for _, v := range values[1:] {
			sb.WriteString("` | `")
			sb.WriteString(v.Value)
		}
		sb.WriteString("`")
	} else if doc := t.Doc(); doc != "" {
		sb.WriteString("\n\n")
		sb.WriteString(doc)
	}
	return sb.String()
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

	if typeDoc := a.Type.markdownDoc(); typeDoc != "" {
		sb.WriteString("**Type:** ")
		sb.WriteString(typeDoc)
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
		{Name: "area", Type: TypeDouble, UsedBy: Node | Cluster, Doc: "Preferred area for node or empty cluster (patchwork layout)"},
		{Name: "arrowhead", Type: TypeArrowType, UsedBy: Edge, Doc: "Style of arrowhead on edge head node"},
		{Name: "arrowsize", Type: TypeDouble, UsedBy: Edge, Doc: "Multiplicative scale factor for arrowheads"},
		{Name: "arrowtail", Type: TypeArrowType, UsedBy: Edge, Doc: "Style of arrowhead on edge tail node"},
		{Name: "bb", Type: TypeRect, UsedBy: Cluster | Graph, Doc: "Bounding box of drawing in points (write-only)"},
		{Name: "beautify", Type: TypeBool, UsedBy: Graph, Doc: "Whether to draw leaf nodes in circle around root (sfdp)"},
		{Name: "bgcolor", Type: TypeColor, UsedBy: Graph | Cluster, Doc: "Canvas background color"},
		{Name: "center", Type: TypeBool, UsedBy: Graph, Doc: "Whether to center drawing in output canvas"},
		{Name: "charset", Type: TypeString, UsedBy: Graph, Doc: "Character encoding for text labels"},
		{Name: "class", Type: TypeString, UsedBy: Edge | Node | Cluster | Graph, Doc: "Classnames for SVG element styling"},
		{Name: "cluster", Type: TypeBool, UsedBy: Cluster | Subgraph, Doc: "Whether subgraph is a cluster"},
		{Name: "clusterrank", Type: TypeClusterMode, UsedBy: Graph, Doc: "Mode for handling clusters (dot layout)"},
		{Name: "color", Type: TypeColor, UsedBy: Edge | Node | Cluster, Doc: "Basic drawing color for graphics"},
		{Name: "colorscheme", Type: TypeString, UsedBy: Edge | Node | Cluster | Graph, Doc: "Color scheme namespace for interpreting color names"},
		{Name: "comment", Type: TypeString, UsedBy: Edge | Node | Graph, Doc: "Comments inserted into output"},
		{Name: "compound", Type: TypeBool, UsedBy: Graph, Doc: "Allow edges between clusters (dot layout)"},
		{Name: "concentrate", Type: TypeBool, UsedBy: Graph, Doc: "Use edge concentrators"},
		{Name: "constraint", Type: TypeBool, UsedBy: Edge, Doc: "Whether edge used in node ranking (dot layout)"},
		{Name: "Damping", Type: TypeDouble, UsedBy: Graph, Doc: "Factor damping force motions (neato layout)"},
		{Name: "decorate", Type: TypeBool, UsedBy: Edge, Doc: "Connect edge label to edge with line"},
		{Name: "defaultdist", Type: TypeDouble, UsedBy: Graph, Doc: "Distance between nodes in separate components (neato)"},
		{Name: "dim", Type: TypeInt, UsedBy: Graph, Doc: "Number of dimensions for layout"},
		{Name: "dimen", Type: TypeInt, UsedBy: Graph, Doc: "Number of dimensions for rendering"},
		{Name: "dir", Type: TypeDirType, UsedBy: Edge, Doc: "Edge type for drawing arrowheads"},
		{Name: "diredgeconstraints", Type: TypeString, UsedBy: Graph, Doc: "Constrain edges to point downwards (neato)"},
		{Name: "distortion", Type: TypeDouble, UsedBy: Node, Doc: "Distortion factor for polygon shapes"},
		{Name: "dpi", Type: TypeDouble, UsedBy: Graph, Doc: "Expected pixels per inch on display device"},
		{Name: "edgehref", Type: TypeEscString, UsedBy: Edge, Doc: "Synonym for edgeURL"},
		{Name: "edgetarget", Type: TypeEscString, UsedBy: Edge, Doc: "Browser window for edgeURL link"},
		{Name: "edgetooltip", Type: TypeEscString, UsedBy: Edge, Doc: "Tooltip on non-label part of edge"},
		{Name: "edgeURL", Type: TypeEscString, UsedBy: Edge, Doc: "Link for non-label parts of edge"},
		{Name: "epsilon", Type: TypeDouble, UsedBy: Graph, Doc: "Terminating condition (neato layout)"},
		{Name: "esep", Type: TypeAddDouble, UsedBy: Graph, Doc: "Margin around polygons for spline edge routing"},
		{Name: "fillcolor", Type: TypeColor, UsedBy: Node | Edge | Cluster, Doc: "Color for filling node or cluster background"},
		{Name: "fixedsize", Type: TypeBool, UsedBy: Node, Doc: "Use specified width/height for node size"},
		{Name: "fontcolor", Type: TypeColor, UsedBy: Edge | Node | Graph | Cluster, Doc: "Color used for text"},
		{Name: "fontname", Type: TypeString, UsedBy: Edge | Node | Graph | Cluster, Doc: "Font used for text"},
		{Name: "fontnames", Type: TypeString, UsedBy: Graph, Doc: "Control fontname representation in SVG"},
		{Name: "fontpath", Type: TypeString, UsedBy: Graph, Doc: "Directory list for bitmap font search"},
		{Name: "fontsize", Type: TypeDouble, UsedBy: Edge | Node | Graph | Cluster, Doc: "Font size in points"},
		{Name: "forcelabels", Type: TypeBool, UsedBy: Graph, Doc: "Force placement of all xlabels"},
		{Name: "gradientangle", Type: TypeInt, UsedBy: Node | Cluster | Graph, Doc: "Angle of gradient fill"},
		{Name: "group", Type: TypeString, UsedBy: Node, Doc: "Name for node group with bundled edges (dot)"},
		{Name: "head_lp", Type: TypePoint, UsedBy: Edge, Doc: "Center position of edge head label (write-only)"},
		{Name: "headclip", Type: TypeBool, UsedBy: Edge, Doc: "Clip edge head to node boundary"},
		{Name: "headhref", Type: TypeEscString, UsedBy: Edge, Doc: "Synonym for headURL"},
		{Name: "headlabel", Type: TypeLblString, UsedBy: Edge, Doc: "Text label near head of edge"},
		{Name: "headport", Type: TypePortPos, UsedBy: Edge, Doc: "Where on head node to attach edge"},
		{Name: "headtarget", Type: TypeEscString, UsedBy: Edge, Doc: "Browser window for headURL link"},
		{Name: "headtooltip", Type: TypeEscString, UsedBy: Edge, Doc: "Tooltip on edge head"},
		{Name: "headURL", Type: TypeEscString, UsedBy: Edge, Doc: "Link for edge head label"},
		{Name: "height", Type: TypeDouble, UsedBy: Node, Doc: "Height of node in inches"},
		{Name: "href", Type: TypeEscString, UsedBy: Graph | Cluster | Node | Edge, Doc: "Synonym for URL"},
		{Name: "id", Type: TypeEscString, UsedBy: Graph | Cluster | Node | Edge, Doc: "Identifier for graph objects"},
		{Name: "image", Type: TypeString, UsedBy: Node, Doc: "File containing image for node"},
		{Name: "imagepath", Type: TypeString, UsedBy: Graph, Doc: "Directories to search for image files"},
		{Name: "imagepos", Type: TypeString, UsedBy: Node, Doc: "Position of image within node"},
		{Name: "imagescale", Type: TypeString, UsedBy: Node, Doc: "How image fills containing node"},
		{Name: "inputscale", Type: TypeDouble, UsedBy: Graph, Doc: "Scales input positions to convert length units"},
		{Name: "K", Type: TypeDouble, UsedBy: Graph | Cluster, Doc: "Spring constant for virtual physical model"},
		{Name: "label", Type: TypeLblString, UsedBy: Edge | Node | Graph | Cluster, Doc: "Text label attached to objects"},
		{Name: "label_scheme", Type: TypeInt, UsedBy: Graph, Doc: "Treat special nodes as edge labels (sfdp)"},
		{Name: "labelangle", Type: TypeDouble, UsedBy: Edge, Doc: "Angle in degrees of head/tail edge labels"},
		{Name: "labeldistance", Type: TypeDouble, UsedBy: Edge, Doc: "Scaling factor for head/tail label distance"},
		{Name: "labelfloat", Type: TypeBool, UsedBy: Edge, Doc: "Allow edge labels less constrained in position"},
		{Name: "labelfontcolor", Type: TypeColor, UsedBy: Edge, Doc: "Color for headlabel and taillabel"},
		{Name: "labelfontname", Type: TypeString, UsedBy: Edge, Doc: "Font for headlabel and taillabel"},
		{Name: "labelfontsize", Type: TypeDouble, UsedBy: Edge, Doc: "Font size for headlabel and taillabel"},
		{Name: "labelhref", Type: TypeEscString, UsedBy: Edge, Doc: "Synonym for labelURL"},
		{Name: "labeljust", Type: TypeString, UsedBy: Graph | Cluster, Doc: "Justification for graph/cluster labels"},
		{Name: "labelloc", Type: TypeString, UsedBy: Node | Graph | Cluster, Doc: "Vertical placement of labels"},
		{Name: "labeltarget", Type: TypeEscString, UsedBy: Edge, Doc: "Browser window for labelURL links"},
		{Name: "labeltooltip", Type: TypeEscString, UsedBy: Edge, Doc: "Tooltip on edge label"},
		{Name: "labelURL", Type: TypeEscString, UsedBy: Edge, Doc: "Link for edge label"},
		{Name: "landscape", Type: TypeBool, UsedBy: Graph, Doc: "Render graph in landscape mode"},
		{Name: "layer", Type: TypeLayerRange, UsedBy: Edge | Node | Cluster, Doc: "Specifies layers for object presence"},
		{Name: "layerlistsep", Type: TypeString, UsedBy: Graph, Doc: "Separator for layerRange splitting"},
		{Name: "layers", Type: TypeLayerList, UsedBy: Graph, Doc: "Linearly ordered list of layer names"},
		{Name: "layerselect", Type: TypeLayerRange, UsedBy: Graph, Doc: "Selects layers to be emitted"},
		{Name: "layersep", Type: TypeString, UsedBy: Graph, Doc: "Separator for layers attribute splitting"},
		{Name: "layout", Type: TypeLayout, UsedBy: Graph, Doc: "Which layout engine to use"},
		{Name: "len", Type: TypeDouble, UsedBy: Edge, Doc: "Preferred edge length in inches"},
		{Name: "levels", Type: TypeInt, UsedBy: Graph, Doc: "Levels allowed in multilevel scheme (sfdp)"},
		{Name: "levelsgap", Type: TypeDouble, UsedBy: Graph, Doc: "Strictness of neato level constraints"},
		{Name: "lhead", Type: TypeString, UsedBy: Edge, Doc: "Logical head of edge (dot layout)"},
		{Name: "lheight", Type: TypeDouble, UsedBy: Graph | Cluster, Doc: "Height of graph/cluster label (write-only)"},
		{Name: "linelength", Type: TypeInt, UsedBy: Graph, Doc: "String length before overflow to next line"},
		{Name: "lp", Type: TypePoint, UsedBy: Edge | Graph | Cluster, Doc: "Label center position (write-only)"},
		{Name: "ltail", Type: TypeString, UsedBy: Edge, Doc: "Logical tail of edge (dot layout)"},
		{Name: "lwidth", Type: TypeDouble, UsedBy: Graph | Cluster, Doc: "Width of graph/cluster label (write-only)"},
		{Name: "margin", Type: TypeDouble, UsedBy: Node | Cluster | Graph, Doc: "Margin around canvas or node content"},
		{Name: "maxiter", Type: TypeInt, UsedBy: Graph, Doc: "Number of iterations for layout"},
		{Name: "mclimit", Type: TypeDouble, UsedBy: Graph, Doc: "Scale factor for mincross edge crossing minimizer"},
		{Name: "mindist", Type: TypeDouble, UsedBy: Graph, Doc: "Minimum separation between all nodes (circo)"},
		{Name: "minlen", Type: TypeInt, UsedBy: Edge, Doc: "Minimum edge length by rank difference (dot)"},
		{Name: "mode", Type: TypeString, UsedBy: Graph, Doc: "Technique for layout optimization (neato)"},
		{Name: "model", Type: TypeString, UsedBy: Graph, Doc: "Distance matrix computation method (neato)"},
		{Name: "newrank", Type: TypeBool, UsedBy: Graph, Doc: "Use single global ranking, ignoring clusters (dot)"},
		{Name: "nodesep", Type: TypeDouble, UsedBy: Graph, Doc: "Minimum space between adjacent nodes"},
		{Name: "nojustify", Type: TypeBool, UsedBy: Graph | Cluster | Node | Edge, Doc: "Justify multiline text vs previous line"},
		{Name: "normalize", Type: TypeDouble, UsedBy: Graph, Doc: "Normalize final layout coordinates"},
		{Name: "notranslate", Type: TypeBool, UsedBy: Graph, Doc: "Avoid translating layout to origin (neato)"},
		{Name: "nslimit", Type: TypeDouble, UsedBy: Graph, Doc: "Iterations in network simplex (dot)"},
		{Name: "nslimit1", Type: TypeDouble, UsedBy: Graph, Doc: "Iterations in network simplex for ranking (dot)"},
		{Name: "oneblock", Type: TypeBool, UsedBy: Graph, Doc: "Draw circo graphs around one circle"},
		{Name: "ordering", Type: TypeString, UsedBy: Graph | Node, Doc: "Constrain left-to-right edge ordering (dot)"},
		{Name: "orientation", Type: TypeString, UsedBy: Node | Graph, Doc: "Node rotation angle or graph orientation"},
		{Name: "outputorder", Type: TypeOutputMode, UsedBy: Graph, Doc: "Order for drawing nodes and edges"},
		{Name: "overlap", Type: TypeString, UsedBy: Graph, Doc: "Remove or determine node overlaps"},
		{Name: "overlap_scaling", Type: TypeDouble, UsedBy: Graph, Doc: "Scale layout to reduce node overlap"},
		{Name: "overlap_shrink", Type: TypeBool, UsedBy: Graph, Doc: "Compress pass for overlap removal"},
		{Name: "pack", Type: TypeBool, UsedBy: Graph, Doc: "Layout components separately then pack"},
		{Name: "packmode", Type: TypePackMode, UsedBy: Graph, Doc: "How connected components should be packed"},
		{Name: "pad", Type: TypeDouble, UsedBy: Graph, Doc: "Inches extending drawing area around graph"},
		{Name: "page", Type: TypeDouble, UsedBy: Graph, Doc: "Width and height of output pages"},
		{Name: "pagedir", Type: TypePagedir, UsedBy: Graph, Doc: "Order in which pages are emitted"},
		{Name: "pencolor", Type: TypeColor, UsedBy: Cluster, Doc: "Color for cluster bounding box"},
		{Name: "penwidth", Type: TypeDouble, UsedBy: Cluster | Node | Edge, Doc: "Width of pen for drawing lines/curves"},
		{Name: "peripheries", Type: TypeInt, UsedBy: Node | Cluster, Doc: "Number of peripheries in shapes/boundaries"},
		{Name: "pin", Type: TypeBool, UsedBy: Node, Doc: "Keep node at input position (neato, fdp)"},
		{Name: "pos", Type: TypePoint, UsedBy: Edge | Node, Doc: "Position of node or spline control points"},
		{Name: "quadtree", Type: TypeQuadType, UsedBy: Graph, Doc: "Quadtree scheme for layout (sfdp)"},
		{Name: "quantum", Type: TypeDouble, UsedBy: Graph, Doc: "Round node label dimensions to quantum multiples"},
		{Name: "radius", Type: TypeDouble, UsedBy: Edge, Doc: "Radius of rounded corners on orthogonal edges"},
		{Name: "rank", Type: TypeRankType, UsedBy: Subgraph, Doc: "Rank constraints on subgraph nodes (dot)"},
		{Name: "rankdir", Type: TypeRankdir, UsedBy: Graph, Doc: "Sets direction of graph layout (dot)"},
		{Name: "ranksep", Type: TypeDouble, UsedBy: Graph, Doc: "Specifies separation between ranks"},
		{Name: "ratio", Type: TypeDouble, UsedBy: Graph, Doc: "Aspect ratio for drawing"},
		{Name: "rects", Type: TypeRect, UsedBy: Node, Doc: "Rectangles for record fields (write-only)"},
		{Name: "regular", Type: TypeBool, UsedBy: Node, Doc: "Force polygon to be regular"},
		{Name: "remincross", Type: TypeBool, UsedBy: Graph, Doc: "Run edge crossing minimization twice (dot)"},
		{Name: "repulsiveforce", Type: TypeDouble, UsedBy: Graph, Doc: "Power of repulsive force (sfdp)"},
		{Name: "resolution", Type: TypeDouble, UsedBy: Graph, Doc: "Synonym for dpi"},
		{Name: "root", Type: TypeString, UsedBy: Graph | Node, Doc: "Nodes for layout center (twopi, circo)"},
		{Name: "rotate", Type: TypeInt, UsedBy: Graph, Doc: "Sets drawing orientation to landscape"},
		{Name: "rotation", Type: TypeDouble, UsedBy: Graph, Doc: "Rotate final layout counter-clockwise (sfdp)"},
		{Name: "samehead", Type: TypeString, UsedBy: Edge, Doc: "Aim edges at same head point (dot)"},
		{Name: "sametail", Type: TypeString, UsedBy: Edge, Doc: "Aim edges at same tail point (dot)"},
		{Name: "samplepoints", Type: TypeInt, UsedBy: Node, Doc: "Points used for circle/ellipse node"},
		{Name: "scale", Type: TypeDouble, UsedBy: Graph, Doc: "Scale layout by factor after initial layout"},
		{Name: "searchsize", Type: TypeInt, UsedBy: Graph, Doc: "Max edges to search for minimum cut (dot)"},
		{Name: "sep", Type: TypeAddDouble, UsedBy: Graph, Doc: "Margin around nodes when removing overlap"},
		{Name: "shape", Type: TypeShape, UsedBy: Node, Doc: "Shape of a node"},
		{Name: "shapefile", Type: TypeString, UsedBy: Node, Doc: "File with user-supplied node content"},
		{Name: "showboxes", Type: TypeInt, UsedBy: Edge | Node | Graph, Doc: "Print guide boxes for debugging (dot)"},
		{Name: "sides", Type: TypeInt, UsedBy: Node, Doc: "Number of sides for polygon shape"},
		{Name: "size", Type: TypeDouble, UsedBy: Graph, Doc: "Maximum width and height of drawing"},
		{Name: "skew", Type: TypeDouble, UsedBy: Node, Doc: "Skew factor for polygon shapes"},
		{Name: "smoothing", Type: TypeSmoothType, UsedBy: Graph, Doc: "Post-processing step for node distribution (sfdp)"},
		{Name: "sortv", Type: TypeInt, UsedBy: Graph | Cluster | Node, Doc: "Sort order for component packing"},
		{Name: "splines", Type: TypeString, UsedBy: Graph, Doc: "How edges are represented"},
		{Name: "start", Type: TypeStartType, UsedBy: Graph, Doc: "Parameter for initial node layout"},
		{Name: "style", Type: TypeStyle, UsedBy: Edge | Node | Cluster | Graph, Doc: "Style information for graph components"},
		{Name: "stylesheet", Type: TypeString, UsedBy: Graph, Doc: "XML style sheet for SVG output"},
		{Name: "tail_lp", Type: TypePoint, UsedBy: Edge, Doc: "Position of edge tail label (write-only)"},
		{Name: "tailclip", Type: TypeBool, UsedBy: Edge, Doc: "Clip edge tail to node boundary"},
		{Name: "tailhref", Type: TypeEscString, UsedBy: Edge, Doc: "Synonym for tailURL"},
		{Name: "taillabel", Type: TypeLblString, UsedBy: Edge, Doc: "Text label near tail of edge"},
		{Name: "tailport", Type: TypePortPos, UsedBy: Edge, Doc: "Where on tail node to attach edge"},
		{Name: "tailtarget", Type: TypeEscString, UsedBy: Edge, Doc: "Browser window for tailURL link"},
		{Name: "tailtooltip", Type: TypeEscString, UsedBy: Edge, Doc: "Tooltip on edge tail"},
		{Name: "tailURL", Type: TypeEscString, UsedBy: Edge, Doc: "Link for edge tail label"},
		{Name: "target", Type: TypeEscString, UsedBy: Edge | Node | Graph | Cluster, Doc: "Browser window for object URL"},
		{Name: "TBbalance", Type: TypeString, UsedBy: Graph, Doc: "Move floating nodes to min/max rank (dot)"},
		{Name: "tooltip", Type: TypeEscString, UsedBy: Node | Edge | Cluster | Graph, Doc: "Tooltip text on hover"},
		{Name: "truecolor", Type: TypeBool, UsedBy: Graph, Doc: "Use truecolor or palette for bitmap rendering"},
		{Name: "URL", Type: TypeEscString, UsedBy: Edge | Node | Graph | Cluster, Doc: "Hyperlinks in device-dependent output"},
		{Name: "vertices", Type: TypePointList, UsedBy: Node, Doc: "Polygon vertex coordinates (write-only)"},
		{Name: "viewport", Type: TypeViewPort, UsedBy: Graph, Doc: "Clipping window on final drawing"},
		{Name: "voro_margin", Type: TypeDouble, UsedBy: Graph, Doc: "Tuning margin for Voronoi technique"},
		{Name: "weight", Type: TypeInt, UsedBy: Edge, Doc: "Weight of edge"},
		{Name: "width", Type: TypeDouble, UsedBy: Node, Doc: "Width of node in inches"},
		{Name: "xdotversion", Type: TypeString, UsedBy: Graph, Doc: "Version of xdot used in output"},
		{Name: "xlabel", Type: TypeLblString, UsedBy: Edge | Node, Doc: "External label for node or edge"},
		{Name: "xlp", Type: TypePoint, UsedBy: Node | Edge, Doc: "Position of exterior label (write-only)"},
		{Name: "z", Type: TypeDouble, UsedBy: Node, Doc: "Z-coordinate for 3D layouts"},
	}
	slices.SortFunc(attributes, func(a, b Attribute) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range attributes {
		attributes[i].MarkdownDoc = attributes[i].markdownDoc()
	}

	return attributes
}()
