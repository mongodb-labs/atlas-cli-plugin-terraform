package hcl

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// MoveAttr deletes an attribute from fromBody and adds it to toBody.
func MoveAttr(fromBody, toBody *hclwrite.Body, fromAttrName, toAttrName, errPrefix string) error {
	tokens, err := PopAttr(fromBody, fromAttrName, errPrefix)
	if err == nil {
		toBody.SetAttributeRaw(toAttrName, tokens)
	}
	return err
}

// PopAttr deletes an attribute and returns it value.
func PopAttr(body *hclwrite.Body, attrName, errPrefix string) (hclwrite.Tokens, error) {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return nil, fmt.Errorf("%s: attribute %s not found", errPrefix, attrName)
	}
	tokens := attr.Expr().BuildTokens(nil)
	body.RemoveAttribute(attrName)
	return tokens, nil
}

// SetAttrExpr sets an attribute to an expression (possibly with interpolations) without quotes.
func SetAttrExpr(body *hclwrite.Body, attrName, expresion string) {
	tokens := hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte(expresion)}}
	body.SetAttributeRaw(attrName, tokens)
}

// SetAttrInt sets an attribute to a number.
func SetAttrInt(body *hclwrite.Body, attrName string, number int) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenNumberLit, Bytes: []byte(strconv.Itoa(number))},
	}
	body.SetAttributeRaw(attrName, tokens)
}

// GetAttrInt tries to get an attribute value as an int.
func GetAttrInt(attr *hclwrite.Attribute, errPrefix string) (int, error) {
	expr, diags := hclsyntax.ParseExpression(attr.Expr().BuildTokens(nil).Bytes(), "", hcl.InitialPos)
	if diags.HasErrors() {
		return 0, fmt.Errorf("%s: failed to parse number: %s", errPrefix, diags.Error())
	}
	val, diags := expr.Value(nil)
	if diags.HasErrors() {
		return 0, fmt.Errorf("%s: failed to evaluate number: %s", errPrefix, diags.Error())
	}
	if !val.Type().Equals(cty.Number) {
		return 0, fmt.Errorf("%s: attribute is not a number", errPrefix)
	}
	num, _ := val.AsBigFloat().Int64()
	return int(num), nil
}

// GetAttrString tries to get an attribute value as a string.
func GetAttrString(attr *hclwrite.Attribute, errPrefix string) (string, error) {
	expr, diags := hclsyntax.ParseExpression(attr.Expr().BuildTokens(nil).Bytes(), "", hcl.InitialPos)
	if diags.HasErrors() {
		return "", fmt.Errorf("%s: failed to parse string: %s", errPrefix, diags.Error())
	}
	val, diags := expr.Value(nil)
	if diags.HasErrors() {
		return "", fmt.Errorf("%s: failed to evaluate string: %s", errPrefix, diags.Error())
	}
	if !val.Type().Equals(cty.String) {
		return "", fmt.Errorf("%s: attribute is not a string", errPrefix)
	}
	return val.AsString(), nil
}

// TokensArray creates an array of objects.
func TokensArray(bodies []*hclwrite.Body) hclwrite.Tokens {
	ret := hclwrite.Tokens{tokenNewLine}
	tokens := make([]hclwrite.Tokens, 0)
	for i := range bodies {
		tokens = append(tokens, TokensObject(bodies[i]))
	}
	ret = append(ret, joinTokens(tokens...)...)
	ret = append(ret, tokenNewLine)
	return encloseBrackets(ret)
}

// TokensArraySingle creates an array of one object.
func TokensArraySingle(body *hclwrite.Body) hclwrite.Tokens {
	return TokensArray([]*hclwrite.Body{body})
}

// TokensObject creates an object.
func TokensObject(body *hclwrite.Body) hclwrite.Tokens {
	tokens := hclwrite.Tokens{tokenNewLine}
	tokens = append(tokens, RemoveLeadingNewline(body.BuildTokens(nil))...)
	return encloseBraces(tokens)
}

// TokensFromExpr creates the tokens for an expression provided as a string.
func TokensFromExpr(expr string) hclwrite.Tokens {
	return hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte(expr)}}
}

// TokensObjectFromExpr creates an object with an expression.
func TokensObjectFromExpr(expr string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{tokenNewLine}
	tokens = append(tokens, TokensFromExpr(expr)...)
	tokens = append(tokens, tokenNewLine)
	return encloseBraces(tokens)
}

// TokensFuncMerge created the tokens for the HCL merge function.
func TokensFuncMerge(tokens ...hclwrite.Tokens) hclwrite.Tokens {
	params := hclwrite.Tokens{tokenNewLine}
	params = append(params, joinTokens(tokens...)...)
	params = append(params, tokenNewLine)
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte("merge")}}
	return append(ret, encloseParens(params)...)
}

// RemoveLeadingNewline removes the first newline if it exists to make the output prettier.
func RemoveLeadingNewline(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) > 0 && tokens[0].Type == hclsyntax.TokenNewline {
		return tokens[1:]
	}
	return tokens
}

// AppendComment adds a comment at the end of the body.
func AppendComment(body *hclwrite.Body, comment string) {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("# " + comment + "\n")},
	}
	body.AppendUnstructuredTokens(tokens)
}

// GetParser returns a parser for the given config and checks HCL syntax is valid
func GetParser(config []byte) (*hclwrite.File, error) {
	parser, diags := hclwrite.ParseConfig(config, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse Terraform config file: %s", diags.Error())
	}
	return parser, nil
}

var tokenNewLine = &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")}

// joinTokens joins multiple tokens with commas and newlines.
func joinTokens(tokens ...hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{}
	for i := range tokens {
		ret = append(ret, tokens[i]...)
		if i < len(tokens)-1 {
			ret = append(ret,
				&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")},
				tokenNewLine)
		}
	}
	return ret
}

// encloseParens encloses tokens with parentheses, ( ).
func encloseParens(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOParen, Bytes: []byte("(")}}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
}

// encloseBraces encloses tokens with curly braces, { }.
func encloseBraces(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")}}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
}

// encloseBrackets encloses tokens with square brackets, [ ].
func encloseBrackets(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")}}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
}
