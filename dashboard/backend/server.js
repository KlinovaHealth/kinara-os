'use strict';

require('dotenv').config();
const express = require('express');
const { graphqlHTTP } = require('express-graphql');
const { buildSchema } = require('graphql');
const cors = require('cors');

const HEALTH_ANALYTICS_URL = process.env.HEALTH_ANALYTICS_URL || 'http://health-analytics-service:8110';
const COMPLIANCE_URL = process.env.COMPLIANCE_URL || 'http://health-compliance-service:8111';
const PATIENT_URL = process.env.PATIENT_URL || 'http://patient-service:8081';
const JWT_SECRET = process.env.JWT_SECRET || 'dev-secret';

const schema = buildSchema(`
  type DiseaseReport {
    id: String!
    clinic_id: String!
    country: String!
    region: String
    icd10_code: String!
    disease_name: String!
    case_count: Int!
    period: String!
    severity: String!
    created_at: String!
  }

  type OutbreakAlert {
    id: String!
    alert_ref: String!
    clinic_id: String!
    country: String!
    disease_name: String!
    case_count: Int!
    status: String!
    detected_at: String!
  }

  type ClinicMetric {
    id: String!
    clinic_id: String!
    country: String!
    total_patients: Int!
    avg_visit_minutes: Float!
    referral_success_rate: Float!
    patient_outcome_improved: Int!
    cost_per_visit_usd: Float!
    period: String!
  }

  type ImpactSummary {
    country: String!
    total_patients: Int!
    outcome_improvement_rate: Float!
    avg_cost_per_visit_usd: Float!
    active_outbreaks: Int!
    total_clinics: Int!
    generated_at: String!
  }

  type ComplianceStatus {
    service: String!
    encrypted_fields: Int!
    total_fields: Int!
    algorithm: String!
    is_compliant: Boolean!
  }

  type DashboardSummary {
    country: String!
    impact: ImpactSummary
    active_outbreaks: Int!
    compliance_score: Float!
    last_updated: String!
  }

  type Query {
    diseases(country: String, icd10_code: String): [DiseaseReport!]!
    outbreaks(country: String): [OutbreakAlert!]!
    clinicMetrics(clinic_id: String!): [ClinicMetric!]!
    impact(country: String): ImpactSummary!
    complianceStatus: [ComplianceStatus!]!
    dashboardSummary(country: String): DashboardSummary!
  }
`);

async function fetchJSON(url, opts = {}) {
  try {
    const res = await fetch(url, {
      ...opts,
      headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
      signal: AbortSignal.timeout(5000),
    });
    if (!res.ok) return null;
    const body = await res.json();
    return body.data || body;
  } catch {
    return null;
  }
}

const root = {
  diseases: async ({ country, icd10_code }) => {
    const params = new URLSearchParams();
    if (country) params.set('country', country);
    if (icd10_code) params.set('icd10', icd10_code);
    const data = await fetchJSON(`${HEALTH_ANALYTICS_URL}/api/v1/health-analytics/diseases?${params}`);
    return Array.isArray(data) ? data : [];
  },

  outbreaks: async ({ country }) => {
    const params = new URLSearchParams();
    if (country) params.set('country', country);
    const data = await fetchJSON(`${HEALTH_ANALYTICS_URL}/api/v1/health-analytics/outbreaks?${params}`);
    return Array.isArray(data) ? data : [];
  },

  clinicMetrics: async ({ clinic_id }) => {
    const data = await fetchJSON(`${HEALTH_ANALYTICS_URL}/api/v1/health-analytics/clinics/${clinic_id}/metrics`);
    return Array.isArray(data) ? data : [];
  },

  impact: async ({ country = 'TG' }) => {
    const data = await fetchJSON(`${HEALTH_ANALYTICS_URL}/api/v1/health-analytics/impact?country=${country}`);
    return data || {
      country, total_patients: 0, outcome_improvement_rate: 0,
      avg_cost_per_visit_usd: 0, active_outbreaks: 0, total_clinics: 0,
      generated_at: new Date().toISOString(),
    };
  },

  complianceStatus: async () => {
    const data = await fetchJSON(`${COMPLIANCE_URL}/api/v1/compliance/encryption`);
    return Array.isArray(data) ? data : [];
  },

  dashboardSummary: async ({ country = 'TG' }) => {
    const [impact, outbreaks, compliance] = await Promise.all([
      root.impact({ country }),
      root.outbreaks({ country }),
      root.complianceStatus(),
    ]);
    const compliantCount = compliance.filter(s => s.is_compliant).length;
    const complianceScore = compliance.length > 0 ? (compliantCount / compliance.length) * 100 : 100;
    return {
      country,
      impact,
      active_outbreaks: outbreaks.length,
      compliance_score: complianceScore,
      last_updated: new Date().toISOString(),
    };
  },
};

const app = express();
app.use(cors({ origin: '*' }));
app.use(express.json());

// GraphQL endpoint
app.use('/graphql', graphqlHTTP({
  schema,
  rootValue: root,
  graphiql: process.env.NODE_ENV !== 'production',
}));

// REST convenience endpoint for the React frontend
app.get('/api/dashboard/:country', async (req, res) => {
  const country = (req.params.country || 'TG').toUpperCase();
  const summary = await root.dashboardSummary({ country });
  const diseases = await root.diseases({ country });
  const outbreaks = await root.outbreaks({ country });
  res.json({ success: true, data: { summary, diseases, outbreaks } });
});

app.get('/health', (_, res) => res.json({ status: 'ok', service: 'government-health-dashboard' }));

const PORT = process.env.PORT || 8116;
app.listen(PORT, () => {
  console.log(JSON.stringify({ level: 'info', service: 'government-health-dashboard', port: PORT }));
});
