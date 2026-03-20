package liquid


type RangeLookup struct {
	StartObj interface{}
	EndObj   interface{}
}

func NewRangeLookup(start, end interface{}) *RangeLookup {
	return &RangeLookup{StartObj: start, EndObj: end}
}

func ParseRangeLookup(startMarkup, endMarkup string, ss *StringScanner, cache map[string]interface{}) (interface{}, error) {
	startObj, err := ParseExpression(startMarkup, ss, cache)
	if err != nil {
		return nil, err
	}
	endObj, err := ParseExpression(endMarkup, ss, cache)
	if err != nil {
		return nil, err
	}

	if IsEvaluatable(startObj) || IsEvaluatable(endObj) {
		return NewRangeLookup(startObj, endObj), nil
	}

	startInt, err := UtilsToInteger(startObj)
	if err != nil {
		return nil, &SyntaxError{BaseError: BaseError{Message: "Invalid expression type in range expression"}}
	}
	endInt, err := UtilsToInteger(endObj)
	if err != nil {
		return nil, &SyntaxError{BaseError: BaseError{Message: "Invalid expression type in range expression"}}
	}

	// Generar colección completa: [1, 2, 3] en lugar de [1, 3]
	var result []interface{}
	if startInt <= endInt {
		for i := startInt; i <= endInt; i++ {
			result = append(result, i)
		}
	} else {
		// Rango inverso: [3, 1] -> [3, 2, 1]
		for i := startInt; i >= endInt; i-- {
			result = append(result, i)
		}
	}
	return result, nil
}

func IsEvaluatable(obj interface{}) bool {
	if obj == nil {
		return false
	}
	_, ok := obj.(interface{ Evaluate(*Context) interface{} })
	return ok
}


func (r *RangeLookup) Evaluate(ctx *Context) interface{} {
	start := ToInt(ctx.Evaluate(r.StartObj))
	end := ToInt(ctx.Evaluate(r.EndObj))

	// Generar colección completa: [1, 2, 3] en lugar de [1, 3]
	var result []interface{}
	if start <= end {
		for i := start; i <= end; i++ {
			result = append(result, i)
		}
	} else {
		// Rango inverso: [3, 1] -> [3, 2, 1]
		for i := start; i >= end; i-- {
			result = append(result, i)
		}
	}
	return result
}

func ToInt(input interface{}) int {
	i, _ := UtilsToInteger(input)
	return i
}
