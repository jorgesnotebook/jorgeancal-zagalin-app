# Decision Trees & Flowcharts

Visual guides to help you make the right architectural and implementation decisions.

##  Decision Tree Index

1. [Should I add this to backend or frontend?](#1-backend-vs-frontend)
2. [Which storage pattern should I use?](#2-storage-pattern-selection)
3. [How should I test this feature?](#3-testing-strategy)
4. [Should I use direct LLM or backend proxy?](#4-llm-integration-pattern)
5. [Do I need to enter plan mode?](#5-plan-mode-decision)
6. [Which Grafana UI component should I use?](#6-ui-component-selection)
7. [How do I handle user permissions?](#7-permission-handling)

---

## 1. Backend vs Frontend

**Question**: Should I implement this feature in backend (Go) or frontend (React)?

```

 Does it involve sensitive data? 
 (API keys, credentials, etc.)   

             
              YES →  BACKEND (Go)
                       Store in secureJsonData
                       Access via backend only
             
              NO
                
                

 Does it need to call external   
 APIs with authentication?       

             
              YES →  BACKEND (Go)
                       Backend proxy pattern
                       Forward user context
             
              NO
                
                

 Does it need rate limiting or   
 query validation?               

             
              YES →  BACKEND (Go)
                       Implement guardrails
                       Validate before execution
             
              NO
                
                

 Is it pure UI interaction?      
 (buttons, forms, navigation)    

             
              YES →  FRONTEND (React)
                       UI components only
                       No backend needed
             
              NO
                
                

 Does it need complex data       
 transformation or caching?      

             
              YES →  BACKEND (Go)
                       Better performance
                       Share cache across users
             
              NO →  FRONTEND (React)
                       Simple operations
                       User-specific state
```

**Examples**:
-  Frontend: Storing API keys →  Backend: Store in secureJsonData
-  Frontend: Calling external APIs →  Backend: Proxy with auth
-  Frontend: Rendering a form → No backend needed
-  Backend: Rate limiting queries → Security requirement

---

## 2. Storage Pattern Selection

**Question**: How should I store data in this plugin?

```

 What are you storing?           

             
              SENSITIVE DATA →  SECURE BACKEND STORAGE
               (passwords, tokens)   pkg/plugin/storage.go
                                     Use secureJsonData
             
              USER PREFERENCES → Choose based on Grafana version:
               (theme, defaults)     
                                      Grafana 11.5+ →  usePluginUserStorage()
                                                           Automatic fallback
                                                           Built-in encryption
                                     
                                      Grafana < 11.5 →  Custom Backend Storage
                                                            pkg/plugin/storage.go
                                                            localStorage fallback
             
              CONVERSATIONS →  DUAL-TIER STORAGE (This plugin)
               (chat history)     Primary: Backend file storage
                                  Fallback: localStorage
                                  See: pkg/plugin/storage.go
             
              TEMPORARY STATE →  REACT STATE
               (form inputs,       useState/useReducer
                UI toggles)        No persistence needed
             
              CACHED DATA → Choose based on scope:
               (metrics, logs)  
                                 Shared across users →  Backend Cache
                                                           pkg/plugin/context/
                                                           Background refresh
                                
                                 Per-user cache →  React Query
                                                       Frontend caching
                                                       TTL-based
             
              CONFIGURATION →  PLUGIN SETTINGS
                (plugin settings)    jsonData/secureJsonData
                                    Grafana's config system
```

**Decision Matrix**:

| Data Type | Storage | Why |
|-----------|---------|-----|
| API Keys | Backend `secureJsonData` | Encrypted, never exposed |
| User theme | `usePluginUserStorage()` | Per-user, synced |
| Chat history | Dual-tier (backend + localStorage) | Persistent, fallback |
| Form input | React `useState` | Temporary, no persistence |
| Metrics cache | Backend cache | Shared, expensive to fetch |
| Query results | React Query | Per-user, TTL-based |

---

## 3. Testing Strategy

**Question**: What tests should I write for this feature?

```

 What did you build?             

             
              NEW UI COMPONENT → Tests needed:
                                     Unit test (Jest)
                                      - Rendering
                                      - Props
                                      - User interactions
                                    
                                     E2E test (if critical path)
                                       - User workflow
                                       - Integration
             
              NEW BACKEND ENDPOINT → Tests needed:
                                          Unit test (Go)
                                           - Handler logic
                                           - Error cases
                                           - Edge cases
                                         
                                          Integration test
                                           - Full request/response
                                           - Auth/permissions
                                         
                                          E2E test (if user-facing)
                                            - End-to-end flow
             
              NEW PAGE → Tests needed:
                             E2E test (Playwright)
                              - Page loads
                              - Navigation works
                              - Key features work
                            
                             Component unit tests
                               - Individual components
             
              BUG FIX → Tests needed:
                            Regression test
                             - Reproduce the bug
                             - Verify fix
                           
                            Edge case tests
                              - Similar scenarios
             
              REFACTORING → Tests needed:
                                 All existing tests still pass
                                  - No new tests unless
                                  - behavior changed
                                
                                 Coverage maintained
                                   - Don't decrease coverage
```

**Coverage Requirements**:
- Frontend: >70%
- Backend: >80%
- Critical paths: 100%

---

## 4. LLM Integration Pattern

**Question**: Should I use direct LLM calls or backend proxy?

```

 What's your LLM use case?       

             
              SIMPLE CHAT → Is system prompt sensitive?
               (basic Q&A)    
                               YES →  BACKEND PROXY
                                        Hide prompts from user
                                        pkg/plugin/assistant.go
                              
                               NO → Can use direct:
                                         llm.chatCompletions()
                                           @grafana/llm
                                           Frontend only
             
              FUNCTION CALLING →  BACKEND PROXY (Recommended)
               (tools, actions)     Secure tool execution
                                    Rate limiting
                                    Audit logging
                                    See: pkg/plugin/assistant_tools.go
             
              CONTEXT INJECTION →  BACKEND PROXY (Required)
               (metrics, logs)        Backend has datasource access
                                      Inject context securely
                                      See: pkg/plugin/context/
             
              MCP AGENTS → Choose based on complexity:
               (tool loops)    
                                Simple →  Frontend MCP
                                            useMCPClient()
                                            Built-in tools
                               
                                Complex →  Backend MCP
                                              Custom tools
                                              Multi-step workflows
             
              STREAMING → Both patterns support streaming:
                              Frontend: llm.streamChatCompletions()
                              Backend: SSE proxy (this plugin)
```

**This Plugin Uses**: Backend Proxy Pattern
-  Secure system prompts
-  Context injection
-  Function calling
-  Rate limiting
-  Audit logging

**Files**:
- Backend: `pkg/plugin/assistant.go`
- Frontend: `src/services/assistantService.ts`

---

## 5. Plan Mode Decision

**Question**: Should I enter plan mode or just implement?

```

 What are you building?          

             
              TYPO FIX →  NO PLAN MODE
               SIMPLE BUG    Just fix it
                             < 10 lines
             
              NEW FEATURE → Is it complex?
                               
                                Simple →  NO PLAN MODE
                                 (< 3 files)   Just implement
                               
                                Complex →  ENTER PLAN MODE
                                  (3+ files)   Plan architecture
                                               Get approval
             
              REFACTORING → How many files?
                               
                                1-2 files →  NO PLAN MODE
                                                Refactor directly
                               
                                3+ files →  ENTER PLAN MODE
                                               Plan approach
                                               Avoid breaking changes
             
              ARCHITECTURE CHANGE →  ALWAYS ENTER PLAN MODE
                                        Significant impact
                                        Team discussion needed
             
              UNCLEAR REQUIREMENTS →  ENTER PLAN MODE
                                         Clarify approach
                                         Get user input
```

**When to use Plan Mode**:
-  Multiple valid approaches exist
-  Touches 3+ files
-  Architectural decisions needed
-  Unclear requirements

**When NOT to use**:
-  Simple fixes (typos, small bugs)
-  Adding obvious missing tests
-  User gave specific instructions

---

## 6. UI Component Selection

**Question**: Which Grafana UI component should I use?

```

 What UI element do you need?    

             
              BUTTON → <Button>
                          variant="primary|secondary|destructive"
                          size="sm|md|lg"
             
              INPUT FIELD → <Input>
                               <Field label="...">
                                 <Input />
                               </Field>
             
              FORM → <Form>
                        Handle validation
                        Auto-layout
             
              LOADING STATE → <LoadingPlaceholder>
                                 <Spinner>
             
              ERROR MESSAGE → <Alert severity="error">
                                 <ErrorBoundaryAlert>
             
              DATA TABLE → <InteractiveTable>
                              Sorting, filtering
             
              MODAL/DIALOG → <Modal>
                                <ConfirmModal>
             
              DROPDOWN → <Select>
                            <Field label="...">
                              <Select options={...} />
                            </Field>
             
              TABS → <TabsBar>
                        <Tab label="..." />
             
              ICON → <Icon name="...">
                        From @grafana/ui
             
              LAYOUT → <VerticalGroup>
                          <HorizontalGroup>
                          <Stack>
             
              CUSTOM →  DON'T BUILD CUSTOM
                           Always use @grafana/ui
                           Ensures consistency
```

**Golden Rule**: ALWAYS use `@grafana/ui` components
-  Consistent with Grafana
-  Themes supported (light/dark)
-  Accessibility built-in
-  Won't break on Grafana updates

**Never**:
-  Build custom buttons
-  Custom inputs
-  Custom modals
-  Hardcoded colors/spacing

---

## 7. Permission Handling

**Question**: How should I handle user permissions?

```

 What needs permission control?  

             
              PAGE ACCESS → Use plugin.json role:
                               {
                                 "type": "page",
                                 "role": "Admin|Editor|Viewer"
                               }
                               Grafana enforces automatically
             
              UI ELEMENT → Frontend check:
               (button, menu)  import { contextSrv } from '@grafana/runtime';
             
                               if (contextSrv.hasRole('Admin')) {
                                 return <AdminButton />;
                               }
             
              BACKEND ENDPOINT → Extract user and check:
                                     func handler(ctx context.Context) {
                                       user := extractUser(ctx)
             
                                       if user.Role != "Admin" {
                                         return errors.New("forbidden")
                                       }
                                     }
             
              DATASOURCE ACCESS → Use user's context:
                                      Grafana handles automatically
                                      Forward user identity
                                      See: pkg/plugin/query_proxy.go
             
              FEATURE FLAG → Configuration-based:
                                 Settings with per-org control
                                 Enable/disable features
                                 See: pkg/plugin/settings.go
```

**Permission Levels**:
- **Admin**: Full access, can configure
- **Editor**: Can create/edit
- **Viewer**: Read-only access

**Security Rules**:
-  Always check permissions on backend
-  Frontend checks are for UX only
-  Never trust frontend permission checks
-  Log permission denials for audit

---

##  Quick Decision Guide

| Question | Answer |
|----------|--------|
| Sensitive data? | Backend only |
| External API call? | Backend proxy |
| Pure UI? | Frontend only |
| User preferences? | usePluginUserStorage() |
| Chat history? | Dual-tier storage |
| New feature (3+ files)? | Enter plan mode |
| Simple fix? | Just implement |
| Need a button? | Use @grafana/ui Button |
| Custom UI component? |  Never! Use @grafana/ui |
| Check permissions? | Backend + frontend (UX) |

---

##  Related Documentation

- **Architecture**: `.claude/rules/00-getting-started/architecture-tour.md`
- **Common Tasks**: `.claude/rules/00-getting-started/common-tasks.md`
- **KISS Principles**: `.claude/rules/02-development/clean-code.md`
- **Security**: `.claude/rules/01-grafana-standards/security.md`
- **Testing**: `.claude/rules/02-development/testing.md`

---

**Last Updated**: 2026-01-10
**Plugin Version**: 0.0.5

**Pro Tip**: When in doubt, choose the simpler option. You can always add complexity later if needed (KISS principle).
