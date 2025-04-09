# Prompts

## Prompt: Save to file

Analyze all the .go file under /workspaces/terraform-provider-power-platform/internal and find any code clarity, security and other issues that you will find.
For each issue found read the template .github/prompts/ai_bug_report.md and use it to create marddown file in .github/prompts/issues_found folder describing the issue.

1. File name should be markdown file based on template located in .github/prompts/ai_bug_report.md
2. Template file has elements using "<<>>" that should be replaced
3. Validate the markdown file you've create is valid
4. ##Impact part of the issue file should contain severity. Severity should be one of: low, medium, high, critical. Severity should be written in bold font
5. Based on the severity each issue markdown file should in a subfolder:

- severity critical in .github/prompts/issues_found/critical
- severity high in .github/prompts/issues_found/high
- severity medium in .github/prompts/issues_found/medium
- severity low in .github/prompts/issues_found/low

## Prompt: Save as GH Issues

Analyze the file other_field_required_when_value_of_validator.go and find ONE code clarity, security and
other issues that you will find.
For each issue found read the template .github/prompts/ai_bug_report.md and use it to create an issue
markdown describing the issue.

1. Template file has elements using that should be replaced
2. ##lmpact part of the issue file should contain severity. Severity should be one of: low, medium, high,
critical. Severity should be written in bold font
3. Name of the issue should be same as Title template part
4. Labels of the issue should be •tai found" and "bug
5. Type of the issue should be "Bug"
6. Use tools to create new issue in the repositiory
