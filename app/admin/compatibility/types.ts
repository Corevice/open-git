export interface CompatEndpointCheck {
  schema: boolean;
  status_code: boolean;
  headers: boolean;
  pagination: boolean;
}

export interface CompatEndpointDiffItem {
  field: string;
  expected: string;
  actual: string;
}

export interface CompatEndpoint {
  method: string;
  path: string;
  status: "pass" | "fail" | "unimplemented";
  checks?: CompatEndpointCheck;
  diff?: CompatEndpointDiffItem[];
  last_run?: string;
}

export interface CompatCoverage {
  total_endpoints: number;
  passing: number;
  failing: number;
  unimplemented: number;
  rate: number;
}

export interface CompatReport {
  generated_at: string;
  coverage: CompatCoverage;
  endpoints: CompatEndpoint[];
}
