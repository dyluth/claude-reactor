# AI Prompt: [Feature Name]

## 1. Feature Description
<!-- AI: Provide a concise, one-paragraph summary of this feature's purpose and the value it delivers to claude-reactor users. Focus on the developer experience and workflow improvements. -->

## 2. Goal Statement
<!-- AI: Clearly state the primary goal of this feature in a single sentence. What specific developer workflow problem does this solve? e.g., "Enable developers to hot-reload code changes without container restart for faster iteration cycles." -->

## Project Analysis & current state

### Technology & architecture
<!-- AI: Analyze the existing codebase. Identify and list the key technologies, frameworks, and architectural patterns relevant to implementing this feature. Mention relevant files or modules. -->

### current state
<!-- AI: Describe the current state of the application *before* this feature is implemented. What is the existing user flow or functionality that will be changed or built upon? -->

## context & problem definition

### problem statement
<!-- AI: Describe the specific problem or user pain point this feature solves. Be precise. Use the "Who, What, Why" format if helpful. -->

### success criteria
<!-- AI: Define specific, measurable, achievable, relevant, and time-bound (SMART) criteria for success. List the key metrics that will indicate the feature is successful. e.g., "Reduce support tickets related to data export by 25% within 3 months." -->

## technical requirements

### functional requirements
<!-- AI: List the specific functions the system must perform. Use a checklist format. e.g., "- The system must allow users to select a date range." Include documentation and operational requirements:
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running -->

### non-functional requirements
<!-- AI: List the non-functional requirements such as performance (e.g., "API response time < 200ms"), security, scalability, and usability. Always include:
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally -->

### Technical Constraints
<!-- AI: List any technical limitations or constraints that must be considered, such as specific library versions, budget, or hardware limitations. -->

## Data & Database changes

### Data model updates
<!-- AI: Specify any new database tables, columns, or relationships required. Provide schema definitions if possible (e.g., SQL DDL or ORM model code). -->

### Data migration plan
<!-- AI: If the data model is changing, provide a step-by-step plan for migrating existing data. Include any scripts or commands that will be used. If no migration is needed, state "N/A". -->

## API & Backend changes

### Data access pattern
<!-- AI: Describe how the backend will access the data. Will it use an ORM, raw SQL queries, or a repository pattern? -->

### server actions
<!-- AI: Detail the new server-side functions, methods, or services that need to be created. Specify their inputs, outputs, and core logic. -->

### Database queries
<!-- AI: Write the specific database queries (e.g., SQL) or ORM/query builder calls that will be executed. -->

### API Routes
<!-- AI: Define the new or updated API endpoints. Specify the HTTP method, URL path, request body, and expected response format (including status codes). e.g., "POST /api/users/{id}/export" -->

## frontend changes

### New components
<!-- AI: List the new UI components that need to be created (e.g., in React, Vue, etc.). Describe their props, state, and behavior. -->

### Page updates
<!-- AI: Describe the changes to existing pages or views. Which components will be added or modified? -->

## Implementation plan

### phase <X> - <summary>
<!-- AI: Break down the implementation into a sequence of logical steps or phases. For each phase, define the goal and the specific tasks involved. This will be used to generate pull requests. Always include a documentation & operations phase with:
- [ ] Create docs/ structure with README, DEVELOPMENT, DEPLOYMENT, OPERATIONS guides  
- [ ] Implement comprehensive Makefile with all development, testing, and deployment commands
- [ ] Validate <10 minute developer onboarding experience
- [ ] Document troubleshooting procedures and common issues -->

## 5. Testing Strategy
### Unit Tests
<!-- AI: Describe the specific units (functions, methods, components) to be tested and the key scenarios to cover for each. -->
### Integration Tests
<!-- AI: Describe how different parts of the feature will be tested together (e.g., API endpoint with database, frontend component with backend service). -->
### End-to-End (E2E) Tests
<!-- AI: Define the user journeys that will be tested from start to finish. Specify the exact steps for each E2E test. -->

## 6. Security Considerations
### Authentication & Authorization
<!-- AI: Specify the required authentication and authorization checks. Which user roles can access this feature? -->
### Data Validation & Sanitization
<!-- AI: Detail the input validation and data sanitization measures to prevent security vulnerabilities like XSS or SQL injection. -->
### Potential Vulnerabilities
<!-- AI: Identify any potential security risks or attack vectors specific to this feature and propose mitigation strategies. -->

## 7. Rollout & Deployment
### Feature Flags
<!-- AI: Specify if a feature flag is needed. If so, what is the name of the flag and what is the default state? -->
### Deployment Steps
<!-- AI: List any specific steps or commands required for deployment that are outside the standard CI/CD pipeline. -->
### Rollback Plan
<!-- AI: Describe the procedure to safely disable or roll back this feature if issues arise in production. -->

## 8. Open Questions & Assumptions
<!-- AI: List any open questions that need answers before or during development, and document any assumptions made during the planning phase. -->
