package main

// @title       Nexus Interview API
// @version     1.0
// @description REST API for the Nexus AI-driven interview platform.
// @description
// @description Authentication: protected endpoints accept a JWT either as an `Authorization: Bearer {token}` header
// @description or as the HttpOnly `access_token` cookie set by the auth endpoints. In Swagger UI, click Authorize and enter `Bearer {token}`.
// @description
// @description Every response is wrapped in the standard envelope `{ success, message, code, data }`.
// @description On 4xx/5xx responses the `message` field may be masked with a reference ID.

// @contact.name Nexus Team

// @host     localhost:8080
// @BasePath /api/v1
// @schemes  http https

// @securityDefinitions.apikey BearerAuth
// @in          header
// @name        Authorization
// @description Enter the token as: Bearer {token}

// @tag.name        Auth
// @tag.description Token refresh and logout. Login is exposed by the module owning the credentials (admin / candidate).
// @tag.name        Admin
// @tag.description Operational endpoints for administrators.
// @tag.name        Candidates
// @tag.description Manage job applicants invited to take an interview.
// @tag.name        Questions
// @tag.description Manage interview questions attached to a job posting.
// @tag.name        System
// @tag.description Unauthenticated liveness endpoints.
