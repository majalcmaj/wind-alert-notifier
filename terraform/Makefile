.PHONY: init fmt fmt-check validate plan apply destroy state-list

init:
	terraform init -input=false

fmt:
	terraform fmt -recursive

# Same check .github/workflows/terraform.yml runs - fails if any file is unformatted.
fmt-check:
	terraform fmt -check -recursive

validate:
	terraform validate

# terraform.tfvars (gitignored, see terraform.tfvars.example) supplies the
# *_token/admin_* variables automatically - no -var-file needed.
plan:
	terraform plan -input=false

# First apply after adding a new ECR repo also runs that repo's one-time
# seed-image local-exec (needs docker + AWS creds with ECR push rights) so
# the matching lambda can be created.
apply:
	terraform apply -input=false

destroy:
	terraform destroy -input=false

state-list:
	terraform state list
