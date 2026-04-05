package masflowsdk_test

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/mas-soft/masflow/sdk"
)

// SendEmailInput is the input for the sendEmail activity.
type SendEmailInput struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IsHTML  bool   `json:"is_html,omitempty"`
}

// SendEmailOutput is the output of the sendEmail activity.
type SendEmailOutput struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	SentAt    string `json:"sent_at"`
}

func SendEmail(_ context.Context, input SendEmailInput) (SendEmailOutput, error) {
	return SendEmailOutput{
		MessageID: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Status:    "sent",
		SentAt:    time.Now().Format(time.RFC3339),
	}, nil
}

// LogInput is the input for the logEvent activity.
type LogInput struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

func LogEvent(_ context.Context, input LogInput) error {
	return nil
}

func Example() {
	// 1. Define the module
	mod := sdk.NewModule("notifications",
		sdk.WithModuleDescription("Email and SMS notification activities"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleIcon("bell"),
		sdk.WithModuleTaskQueue("notifications-task-queue"),
		sdk.WithModuleAuthor("acme-corp"),
		sdk.WithModuleCategory("notifications"),
		sdk.WithModuleTags("email", "sms", "alerts"),
	)

	// 2. Register activities
	sdk.Register(mod, "sendEmail", SendEmail,
		sdk.WithDescription("Send an email notification"),
		sdk.WithIcon("mail"),
		sdk.WithCategory("email"),
		sdk.WithTags("email", "smtp"),
		sdk.WithDocumentationURL("https://docs.example.com/activities/send-email"),
	)

	sdk.RegisterVoid(mod, "logEvent", LogEvent,
		sdk.WithDescription("Log a workflow event"),
		sdk.WithIcon("file-text"),
		sdk.WithCategory("logging"),
	)

	// 3. Verify registration
	activities := mod.Activities()
	fmt.Printf("Module: %s\n", mod.Name)
	fmt.Printf("Activities registered: %d\n", len(activities))

	if def, ok := mod.GetActivity("sendEmail"); ok {
		fmt.Printf("Activity: %s - %s\n", def.Name, def.Description)
	}

	// 4. To run as a standalone service:
	//    The platform provides Temporal connection details during registration.
	//
	//   runner, err := sdk.NewRunner(mod,
	//       sdk.WithPlatformURL("http://localhost:9999"),
	//   )
	//   if err != nil {
	//       log.Fatal(err)
	//   }
	//   if err := runner.Run(context.Background()); err != nil {
	//       log.Fatal(err)
	//   }

	// Output:
	// Module: notifications
	// Activities registered: 2
	// Activity: sendEmail - Send an email notification
}
