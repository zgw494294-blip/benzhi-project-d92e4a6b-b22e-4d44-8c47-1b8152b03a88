package assessment

func PlanDigest(plan Plan) (string, error) {
	result := AssessPlan(plan)
	if !result.Valid {
		return "", &PlanValidationError{Issues: result.Issues}
	}
	return result.PlanDigest, nil
}
