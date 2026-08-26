package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/assessment"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
)

func runSelfCheck(configuration config) error {
	directory, err := os.MkdirTemp("", "benzhi-certification-selfcheck-")
	if err != nil {
		return fmt.Errorf("创建自检目录: %w", err)
	}
	defer os.RemoveAll(directory)
	app, err := assemble(configuration.Address, directory)
	if err != nil {
		return err
	}
	serveResult := app.serve()
	ctx, cancel := context.WithTimeout(context.Background(), configuration.SelfCheckTimeout)
	defer cancel()
	flowErr := executeSelfCheckFlow(ctx, configuration.Address)
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	shutdownErr := app.shutdown(shutdownContext)
	serveErr := <-serveResult
	if flowErr != nil {
		return fmt.Errorf("HTTP 自检失败: %w", flowErr)
	}
	if shutdownErr != nil {
		return fmt.Errorf("自检关闭失败: %w", shutdownErr)
	}
	if serveErr != nil {
		return fmt.Errorf("自检服务失败: %w", serveErr)
	}
	fmt.Println("自检通过：建档、锁定、有序实测、偏差整改、定向复测、质量冻结、凭据签发与公开校验均成功")
	return nil
}

func executeSelfCheckFlow(ctx context.Context, address string) error {
	client := newSelfCheckClient(address)
	var health map[string]any
	if err := client.get(ctx, "/healthz", &health); err != nil {
		return err
	}
	var value certification.CertificationCase
	err := client.post(ctx, "/api/v1/certification-cases", "certification_engineer", map[string]any{
		"cabinetCode": "SC-CAB-001", "location": "自检实验室 A", "cabinetClass": "Class II", "baselineStatus": "qualified",
	}, &value, http.StatusCreated)
	if err != nil {
		return err
	}
	casePath := "/api/v1/certification-cases/" + value.CaseID
	err = client.post(ctx, casePath+"/plan/lock", "certification_engineer", certification.LockPlanCommand{
		ExpectedVersion: value.Version, StandardCode: "JG/T 292", Revision: "selfcheck-1",
		VelocityRange: assessment.VelocityRange{Min: 0.3, Max: 0.6}, ParticleLimit: 100,
		IntegrityRequired: true, PointOrder: []string{"P1", "P2"},
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	err = client.post(ctx, casePath+"/measurements", "certification_engineer", certification.SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: "P1", Velocity: 0.2, ParticleCount: 80,
		IntegrityPassed: true, EvidenceDigest: "sha256:selfcheck-p1-failed", MeasuredBy: "现场工程师",
		MeasuredAt: time.Now().UTC(), Assignee: "整改工程师", DueAt: time.Now().Add(24 * time.Hour).UTC(),
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	if len(value.Deviations) != 1 {
		return fmt.Errorf("不合格测量未生成唯一偏差")
	}
	deviationID := value.Deviations[0].DeviationID
	err = client.post(ctx, casePath+"/deviations/remediate", "certification_engineer", certification.RemediateCommand{
		ExpectedVersion: value.Version, DeviationID: deviationID, RemediationNote: "完成风机校准并复核安装状态",
		EvidenceDigest: "sha256:selfcheck-remediation", Actor: "整改工程师",
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	err = client.post(ctx, casePath+"/retests", "certification_engineer", certification.RetestCommand{
		ExpectedVersion: value.Version, DeviationID: deviationID, Velocity: 0.45, ParticleCount: 70,
		IntegrityPassed: true, EvidenceDigest: "sha256:selfcheck-p1-retest", MeasuredBy: "现场工程师", MeasuredAt: time.Now().UTC(),
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	err = client.post(ctx, casePath+"/measurements", "certification_engineer", certification.SubmitMeasurementCommand{
		ExpectedVersion: value.Version, PointID: "P2", Velocity: 0.46, ParticleCount: 60,
		IntegrityPassed: true, EvidenceDigest: "sha256:selfcheck-p2", MeasuredBy: "现场工程师", MeasuredAt: time.Now().UTC(),
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	err = client.post(ctx, casePath+"/review", "quality_reviewer", certification.ReviewCommand{
		ExpectedVersion: value.Version, Decision: "approve", Reviewer: "质量复核员", Comment: "测点完整且偏差已闭环",
	}, &value, http.StatusOK)
	if err != nil {
		return err
	}
	var issued credential.ReleaseCredential
	err = client.post(ctx, casePath+"/credentials", "quality_reviewer", certification.IssueCredentialCommand{
		ExpectedVersion: value.Version, IssuedBy: "质量复核员",
	}, &issued, http.StatusCreated)
	if err != nil {
		return err
	}
	var verification credential.Verification
	verifyPath := "/api/v1/credentials/" + issued.CredentialID + "/verify?code=" + url.QueryEscape(issued.VerificationCode)
	if err := client.get(ctx, verifyPath, &verification); err != nil {
		return err
	}
	if !verification.Valid {
		return fmt.Errorf("签发凭据未通过公开校验: %s", verification.Reason)
	}
	return nil
}
