package stdhttp

//----------------------------------------------------------------------------------------------------------------------------//

var css = []byte(`
* {
	margin: 0;
	padding: 0;
	color: #000000;
	outline: 0;
	font-size: 13px;
	font-family: Tahoma, Verdana,  Arial, Helvetica, sans-serif;
	font-style: normal;
	font-variant: normal;
	font-weight: normal;
	line-height: 150%;
	text-align: left;
}


body {
	max-width: 1024px;
	margin: 20px 50px;
	background-color: #f0f0ff
}

.arial, .arial * {
	font-family: Arial, Sans-Serif !important;
}

hr {
	color: #C4C9CE;
	background-color: #C4C9CE;
}

img {
	border: 0px;
}

img.noborder  {
	border: 0px !important;
}

h1, h2, h3, h4, h5, h6, p, center, blockquote, ul, ol, dl {
	margin-bottom: 6px;
}


h1, h2, h3, h4, h5, h6 {
	margin-left: 20px;
}

h1, h2 {
	margin-top: 25px;
}

h3, h4, h5, h6 {
	margin-top: 20px;
}

p, center, blockquote {
	margin-top: 12px;
}

ul, ol, dl {
	margin-top: 12px;
}

h1, h1 *, h2, h2 *, h3, h3 *, h4, h4 *, h5, h5 *, h6, h6 * {
	font-weight: bold;
	font-family: Arial, Sans-Serif;
}

h1, h1 * {
	font-size: 18px;
	text-align: center
}

h2, h2 * {
	font-size: 17px;
}

h3, h3 * {
	font-size: 16px;
}

h4, h4 * {
	font-size: 15px;
}

h5, h5 * {
	font-size: 14px;
}

h6, h6 * {
	font-size: 13px;
}

.top {
	margin-top: 0px;
	padding-top: 0px;
}

blockquote {
	padding-left: 30px;
}

option {
	padding: 2px 3px;
}

.tt, .tt * {
	font-family: monospace !important;
} 

ul.packed li, ol.packed li {
	padding-top: 0 !important;
}

ul li, ol li {
	list-style-position: inside;
}

ul.square li {
	list-style-type: square;
}

ul.circle li {
	list-style-type: circle;
}

ul li, ul.disc li {
	list-style-type: disc;
}

ol li, ol.decimal li {
	list-style-type: decimal;
}

ol.upper_roman li {
	list-style-type: upper-roman;
}

ul.inside li, ol.inside li {
	list-style-position: inside;
}

dl dt, dl dt * {
	font-weight: bold;
}

dl dd {
	padding: 0px 0px 10px 30px;
}

em, italic, .italic, .italic * {
	font-style: italic;
}

strong, .strong, .strong * {
	font-weight: bold;
}

.normal, .normal * {
	font-weight: normal;
}

.attention, .attention * {
	color: red !important;
}

sup.footnote, sup.footnote * {
	font-weight: normal !important;
	color: #CC0000;
}

sup.footnote a, sup.footnote a * {
	text-decoration: none !important;
}

.TODO {
	color: red !important;
	font-size: 20px !important;
}

a * {
	color: #000000
}

a:hover, a:hover * {
	text-decoration:none; color: #515254 !important;
}

table {
	width: 100%;
	border-collapse: collapse;
}

table th {
	text-align: center;
	font-weight: bold;
}

table td {
	text-align: left;
}


table.grd, table.grd2, table.grd3 {
	width: auto;
	border: 1px solid #C4C9CE;
}

table.grd th, table.grd td, table.grd2 th, table.grd2 td, table.grd3 th, table.grd3 td {
	padding: 3px 10px;
	border: 1px solid #C4C9CE;
}

table.grd th, table.grd2 th, table.grd3 th {
	background-color: #DBDDE0;
}

table.grd td, table.grd .even td, table.grd2 td, table.grd2 .even td, table.grd3 td, table.grd3 .even td {
	background-color: #FFFFFF;
}

table.grd .odd td, table.grd2 .odd td, table.grd3 .odd td {
	background-color: #F2F4F7;
}

table.rating_table th, table.rating_table td {
	padding: 3px 5px;
}

table.noborder, table.noborder td, table.noborder th {
	border: none !important;
}

table.noborder td, table.noborder th {
	padding: 1px 10px !important;
}


table.comment {
	border: 1px solid #C4C9CE;
}

table.comment th, table.comment td {
	padding: 2px 4px;
	border: 0;
}

table.comment th {
	background-color: #DBDDE0;
}

table.comment td {
	background-color: #FFFFE0;
}

table.comment th , table.comment th *, table.comment td, table.comment td * {
	font-size: 10px;
	font-family: Arial, Sans-Serif;
}

table.comment th , table.comment th * {
}

table.simple {
	border: 2px solid #000000;
}

table.simple th, table.simple td {
	padding: 4px 10px;
	border: 1px solid #000000;
}


table.condensed {
	border: 0;
}

table.condensed th, table.simple td {
	padding: 0;
	border: 0;
}


table.informer {
	margin: 0;
	border: 0;
	line-height: 120%;
}

table.informer td {
	padding: 4px 5px;
	border: 0;
	background-color: #515F79 !important;
}

table.informer td, table.informer td * {
	color: white;
	font-size: 14px !important;
}

.nobr {
	white-space: nowrap;
}

.left {
	text-align: left !important;
}

.right {
	text-align: right !important;
}

.center {
	text-align: center !important;
}

table.center {
	margin-left: auto !important;
	margin-right: auto !important;
}


.justify {
	text-align: justify !important;
}


small, small * {
	font-size: 9px;
}

big, big * {
	font-size: 15px;
}

.float_left, .float_right {
	margin-bottom: 5px;
}

.float_left {
	margin-right: 10px;
	float: left;
}

.float_right {
	margin-left: 10px;
	float: right;
}


.error .error_header {
	margin-bottom: 10px !important;
	color: red;
	font-size: 15px;
	font-weight: bold;
}

.error .error_header a {
	color: red;
	font-size: 15px;
	font-weight: normal;
}

.error .comment {
	font-weight: bold;
}

.error .comment .url {
	color: red;
	font-size: 15px;
	font-weight: normal;
}


img.btn_16x16 {
	width: 16px;
	height: 16px;
	border: 0;
	vertical-align: middle !important;
}

div.powered {
	margin-top: 10px !important;
	text-align: center !important;
}


input[disabled] {
	color: gray;
	background-color: #FCFCFC;
}


blockquote, q {
	quotes: none;
}

blockquote:before, blockquote:after,
	q:before, q:after {
	content: '';
}


:focus {
	outline: 0;
}


.hidden{
	display:none;
}

table.filters-table {
  width: 1%;
  border: 1px solid #515254;
  background-color: #DBDDE0;
}

table.filters-table th {
  width: 1%;
  white-space: nowrap;
  text-align: left;
  padding: 2px 10px;
}

table.filters-table td {
  width: 1%;
  white-space: nowrap;
  padding: 2px 10px;
}
`)

//----------------------------------------------------------------------------------------------------------------------------//
