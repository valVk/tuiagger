export interface KeyValuePair {
  id: string;
  key: string;
  value: string;
  enabled: boolean;
}

export type HttpMethodType = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' | 'HEAD' | 'OPTIONS';

export interface ManualRequestState {
  method: HttpMethodType;
  path: string;
  queryParams: KeyValuePair[];
  headers: KeyValuePair[];
  body: string;
  bodyType: 'json' | 'raw';
}

export interface ParameterValue {
  name: string;
  in: 'path' | 'query' | 'header';
  value: string;
  required: boolean;
  valid: boolean;
}

export interface RequestState {
  parameters: ParameterValue[];
  body: string | null;
  contentType: string;
}

export interface ResponseState {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  body: string;
  time: number;
  error?: string;
  requestMethod?: string;
  requestUrl?: string;
  requestHeaders?: Record<string, string>;
  requestBody?: string;
}

export interface SavedRequest extends ManualRequestState {
  id: string;
  name: string;
  tag: string;
  createdAt: string;
  updatedAt: string;
}

export interface CustomTag {
  name: string;
  description?: string;
}

export interface SavedRequestsStore {
  version: string;
  requests: SavedRequest[];
  customTags: CustomTag[];
}

// Custom parameter added by user (not in spec)
export interface CustomParameter {
  id: string;
  name: string;
  value: string;
  in: 'query' | 'header' | 'path';
  enabled: boolean;
}

// Overrides for endpoints - stores user's parameter values
export interface EndpointOverride {
  // Parameter values keyed by parameter name
  params: Record<string, string>;
  // Custom parameters added by user (not in spec)
  customParams: CustomParameter[];
  // Parameter names that are disabled (won't be sent)
  disabledParams: string[];
  // Request body if applicable
  body?: string;
  // Override path (if different from spec)
  overridePath?: string;
  // Override method (if different from spec)
  overrideMethod?: string;
  // Last used timestamp
  lastUsed: string;
}

export interface OverridesStore {
  version: string;
  // Keyed by endpoint ID (e.g., "GET /pet/{petId}")
  endpoints: Record<string, EndpointOverride>;
}

export interface AuthStore {
  version: string;
  // scheme name → stored credential value
  credentials: Record<string, string>;
}

export interface Environment {
  name: string;
  variables: Record<string, string>;
}

export interface EnvironmentsStore {
  version: string;
  environments: Environment[];
  activeIndex: number;
}

export interface RequestSpec {
  method: string;
  baseUrl: string;
  /** Path with {param} placeholders already substituted */
  path: string;
  queryParams: KeyValuePair[];
  headerParams: KeyValuePair[];
  body?: string;
  operationSecurity?: import('./openapi.js').SecurityRequirementObject[];
  securitySchemes?: Record<string, import('./openapi.js').SecuritySchemeObject>;
  authCredentials?: Record<string, string>;
}
