package structs

type CreateJobDTO struct {
	CronExpression string `json:"cron_expression"`
	HTTPRequest
}

type UpdateJobDTO struct {
	QueueName string
	CreateJobDTO
}
