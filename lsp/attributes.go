package lsp

import (
	"cmp"
	"slices"
	"strings"
)

// attributeContext represents which DOT elements an attribute can be used with.
type attributeContext uint

const (
	Graph attributeContext = 1 << iota
	Subgraph
	Cluster
	Node
	Edge
)

// String returns the string representation of the attribute context.
// For combined contexts (bitmask), it returns a comma-separated list.
func (c attributeContext) String() string {
	if c == 0 {
		return ""
	}

	// Pre-allocate for all context kinds
	contexts := make([]attributeContext, 0, 5)
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

// attribute represents a Graphviz attribute with its applicable targets and documentation.
type attribute struct {
	name          string
	usedBy        attributeContext
	documentation string
}

// attributes contains all Graphviz attributes from https://graphviz.org/doc/info/attrs.html
var attributes []attribute = func() []attribute {
	attributes := []attribute{
		{"_background", Graph, "Specifies arbitrary background using xdot format strings"},
		{"area", Node | Cluster, "referred area for node or empty cluster (patchwork layout)"},
		{"arrowhead", Edge, "Style of arrowhead on edge head node"},
		{"arrowsize", Edge, "Multiplicative scale factor for arrowheads"},
		{"arrowtail", Edge, "Style of arrowhead on edge tail node"},
		{"bb", Cluster | Graph, "Bounding box of drawing in points (write-only)"},
		{"beautify", Graph, "Whether to draw leaf nodes in circle around root (sfdp)"},
		{"bgcolor", Graph | Cluster, "Canvas background color"},
		{"center", Graph, "Whether to center drawing in output canvas"},
		{"charset", Graph, "Character encoding for text labels"},
		{"class", Edge | Node | Cluster | Graph, "Classnames for SVG element styling"},
		{"cluster", Cluster | Subgraph, "Whether subgraph is a cluster"},
		{"clusterrank", Graph, "Mode for handling clusters (dot layout)"},
		{"color", Edge | Node | Cluster, "Basic drawing color for graphics"},
		{"colorscheme", Edge | Node | Cluster | Graph, "Color scheme namespace for interpreting color names"},
		{"comment", Edge | Node | Graph, "Comments inserted into output"},
		{"compound", Graph, "Allow edges between clusters (dot layout)"},
		{"concentrate", Graph, "Use edge concentrators"},
		{"constraint", Edge, "Whether edge used in node ranking (dot layout)"},
		{"Damping", Graph, "Factor damping force motions (neato layout)"},
		{"decorate", Edge, "Connect edge label to edge with line"},
		{"defaultdist", Graph, "Distance between nodes in separate components (neato)"},
		{"dim", Graph, "Number of dimensions for layout"},
		{"dimen", Graph, "Number of dimensions for rendering"},
		{"dir", Edge, "Edge type for drawing arrowheads"},
		{"diredgeconstraints", Graph, "Constrain edges to point downwards (neato)"},
		{"distortion", Node, "Distortion factor for polygon shapes"},
		{"dpi", Graph, "Expected pixels per inch on display device"},
		{"edgehref", Edge, "Synonym for edgeURL"},
		{"edgetarget", Edge, "Browser window for edgeURL link"},
		{"edgetooltip", Edge, "Tooltip on non-label part of edge"},
		{"edgeURL", Edge, "Link for non-label parts of edge"},
		{"epsilon", Graph, "Terminating condition (neato layout)"},
		{"esep", Graph, "Margin around polygons for spline edge routing"},
		{"fillcolor", Node | Edge | Cluster, "Color for filling node or cluster background"},
		{"fixedsize", Node, "Use specified width/height for node size"},
		{"fontcolor", Edge | Node | Graph | Cluster, "Color used for text"},
		{"fontname", Edge | Node | Graph | Cluster, "Font used for text"},
		{"fontnames", Graph, "Control fontname representation in SVG"},
		{"fontpath", Graph, "Directory list for bitmap font search"},
		{"fontsize", Edge | Node | Graph | Cluster, "Font size in points"},
		{"forcelabels", Graph, "Force placement of all xlabels"},
		{"gradientangle", Node | Cluster | Graph, "Angle of gradient fill"},
		{"group", Node, "Name for node group with bundled edges (dot)"},
		{"head_lp", Edge, "Center position of edge head label (write-only)"},
		{"headclip", Edge, "Clip edge head to node boundary"},
		{"headhref", Edge, "Synonym for headURL"},
		{"headlabel", Edge, "Text label near head of edge"},
		{"headport", Edge, "Where on head node to attach edge"},
		{"headtarget", Edge, "Browser window for headURL link"},
		{"headtooltip", Edge, "Tooltip on edge head"},
		{"headURL", Edge, "Link for edge head label"},
		{"height", Node, "Height of node in inches"},
		{"href", Graph | Cluster | Node | Edge, "Synonym for URL"},
		{"id", Graph | Cluster | Node | Edge, "Identifier for graph objects"},
		{"image", Node, "File containing image for node"},
		{"imagepath", Graph, "Directories to search for image files"},
		{"imagepos", Node, "Position of image within node"},
		{"imagescale", Node, "How image fills containing node"},
		{"inputscale", Graph, "Scales input positions to convert length units"},
		{"K", Graph | Cluster, "Spring constant for virtual physical model"},
		{"label", Edge | Node | Graph | Cluster, "Text label attached to objects"},
		{"label_scheme", Graph, "Treat special nodes as edge labels (sfdp)"},
		{"labelangle", Edge, "Angle in degrees of head/tail edge labels"},
		{"labeldistance", Edge, "Scaling factor for head/tail label distance"},
		{"labelfloat", Edge, "Allow edge labels less constrained in position"},
		{"labelfontcolor", Edge, "Color for headlabel and taillabel"},
		{"labelfontname", Edge, "Font for headlabel and taillabel"},
		{"labelfontsize", Edge, "Font size for headlabel and taillabel"},
		{"labelhref", Edge, "Synonym for labelURL"},
		{"labeljust", Graph | Cluster, "Justification for graph/cluster labels"},
		{"labelloc", Node | Graph | Cluster, "Vertical placement of labels"},
		{"labeltarget", Edge, "Browser window for labelURL links"},
		{"labeltooltip", Edge, "Tooltip on edge label"},
		{"labelURL", Edge, "Link for edge label"},
		{"landscape", Graph, "Render graph in landscape mode"},
		{"layer", Edge | Node | Cluster, "Specifies layers for object presence"},
		{"layerlistsep", Graph, "Separator for layerRange splitting"},
		{"layers", Graph, "Linearly ordered list of layer names"},
		{"layerselect", Graph, "Selects layers to be emitted"},
		{"layersep", Graph, "Separator for layers attribute splitting"},
		{"layout", Graph, "Which layout engine to use"},
		{"len", Edge, "Preferred edge length in inches"},
		{"levels", Graph, "Levels allowed in multilevel scheme (sfdp)"},
		{"levelsgap", Graph, "Strictness of neato level constraints"},
		{"lhead", Edge, "Logical head of edge (dot layout)"},
		{"lheight", Graph | Cluster, "Height of graph/cluster label (write-only)"},
		{"linelength", Graph, "String length before overflow to next line"},
		{"lp", Edge | Graph | Cluster, "Label center position (write-only)"},
		{"ltail", Edge, "Logical tail of edge (dot layout)"},
		{"lwidth", Graph | Cluster, "Width of graph/cluster label (write-only)"},
		{"margin", Node | Cluster | Graph, "Margin around canvas or node content"},
		{"maxiter", Graph, "Number of iterations for layout"},
		{"mclimit", Graph, "Scale factor for mincross edge crossing minimizer"},
		{"mindist", Graph, "Minimum separation between all nodes (circo)"},
		{"minlen", Edge, "Minimum edge length by rank difference (dot)"},
		{"mode", Graph, "Technique for layout optimization (neato)"},
		{"model", Graph, "Distance matrix computation method (neato)"},
		{"newrank", Graph, "Use single global ranking, ignoring clusters (dot)"},
		{"nodesep", Graph, "Minimum space between adjacent nodes"},
		{"nojustify", Graph | Cluster | Node | Edge, "Justify multiline text vs previous line"},
		{"normalize", Graph, "Normalize final layout coordinates"},
		{"notranslate", Graph, "Avoid translating layout to origin (neato)"},
		{"nslimit", Graph, "Iterations in network simplex (dot)"},
		{"nslimit1", Graph, "Iterations in network simplex for ranking (dot)"},
		{"oneblock", Graph, "Draw circo graphs around one circle"},
		{"ordering", Graph | Node, "Constrain left-to-right edge ordering (dot)"},
		{"orientation", Node | Graph, "Node rotation angle or graph orientation"},
		{"outputorder", Graph, "Order for drawing nodes and edges"},
		{"overlap", Graph, "Remove or determine node overlaps"},
		{"overlap_scaling", Graph, "Scale layout to reduce node overlap"},
		{"overlap_shrink", Graph, "Compress pass for overlap removal"},
		{"pack", Graph, "Layout components separately then pack"},
		{"packmode", Graph, "How connected components should be packed"},
		{"pad", Graph, "Inches extending drawing area around graph"},
		{"page", Graph, "Width and height of output pages"},
		{"pagedir", Graph, "Order in which pages are emitted"},
		{"pencolor", Cluster, "Color for cluster bounding box"},
		{"penwidth", Cluster | Node | Edge, "Width of pen for drawing lines/curves"},
		{"peripheries", Node | Cluster, "Number of peripheries in shapes/boundaries"},
		{"pin", Node, "Keep node at input position (neato, fdp)"},
		{"pos", Edge | Node, "Position of node or spline control points"},
		{"quadtree", Graph, "Quadtree scheme for layout (sfdp)"},
		{"quantum", Graph, "Round node label dimensions to quantum multiples"},
		{"radius", Edge, "Radius of rounded corners on orthogonal edges"},
		{"rank", Subgraph, "Rank constraints on subgraph nodes (dot)"},
		{"rankdir", Graph, "Sets direction of graph layout (dot)"},
		{"ranksep", Graph, "Specifies separation between ranks"},
		{"ratio", Graph, "Aspect ratio for drawing"},
		{"rects", Node, "Rectangles for record fields (write-only)"},
		{"regular", Node, "Force polygon to be regular"},
		{"remincross", Graph, "Run edge crossing minimization twice (dot)"},
		{"repulsiveforce", Graph, "Power of repulsive force (sfdp)"},
		{"resolution", Graph, "Synonym for dpi"},
		{"root", Graph | Node, "Nodes for layout center (twopi, circo)"},
		{"rotate", Graph, "Sets drawing orientation to landscape"},
		{"rotation", Graph, "Rotate final layout counter-clockwise (sfdp)"},
		{"samehead", Edge, "Aim edges at same head point (dot)"},
		{"sametail", Edge, "Aim edges at same tail point (dot)"},
		{"samplepoints", Node, "Points used for circle/ellipse node"},
		{"scale", Graph, "Scale layout by factor after initial layout"},
		{"searchsize", Graph, "Max edges to search for minimum cut (dot)"},
		{"sep", Graph, "Margin around nodes when removing overlap"},
		{"shape", Node, "Shape of a node"},
		{"shapefile", Node, "File with user-supplied node content"},
		{"showboxes", Edge | Node | Graph, "Print guide boxes for debugging (dot)"},
		{"sides", Node, "Number of sides for polygon shape"},
		{"size", Graph, "Maximum width and height of drawing"},
		{"skew", Node, "Skew factor for polygon shapes"},
		{"smoothing", Graph, "Post-processing step for node distribution (sfdp)"},
		{"sortv", Graph | Cluster | Node, "Sort order for component packing"},
		{"splines", Graph, "How edges are represented"},
		{"start", Graph, "Parameter for initial node layout"},
		{"style", Edge | Node | Cluster | Graph, "Style information for graph components"},
		{"stylesheet", Graph, "XML style sheet for SVG output"},
		{"tail_lp", Edge, "Position of edge tail label (write-only)"},
		{"tailclip", Edge, "Clip edge tail to node boundary"},
		{"tailhref", Edge, "Synonym for tailURL"},
		{"taillabel", Edge, "Text label near tail of edge"},
		{"tailport", Edge, "Where on tail node to attach edge"},
		{"tailtarget", Edge, "Browser window for tailURL link"},
		{"tailtooltip", Edge, "Tooltip on edge tail"},
		{"tailURL", Edge, "Link for edge tail label"},
		{"target", Edge | Node | Graph | Cluster, "Browser window for object URL"},
		{"TBbalance", Graph, "Move floating nodes to min/max rank (dot)"},
		{"tooltip", Node | Edge | Cluster | Graph, "Tooltip text on hover"},
		{"truecolor", Graph, "Use truecolor or palette for bitmap rendering"},
		{"URL", Edge | Node | Graph | Cluster, "Hyperlinks in device-dependent output"},
		{"vertices", Node, "Polygon vertex coordinates (write-only)"},
		{"viewport", Graph, "Clipping window on final drawing"},
		{"voro_margin", Graph, "Tuning margin for Voronoi technique"},
		{"weight", Edge, "Weight of edge"},
		{"width", Node, "Width of node in inches"},
		{"xdotversion", Graph, "Version of xdot used in output"},
		{"xlabel", Edge | Node, "External label for node or edge"},
		{"xlp", Node | Edge, "Position of exterior label (write-only)"},
		{"z", Node, "Z-coordinate for 3D layouts"},
	}
	slices.SortFunc(attributes, func(a, b attribute) int {
		return cmp.Compare(a.name, b.name)
	})

	return attributes
}()
