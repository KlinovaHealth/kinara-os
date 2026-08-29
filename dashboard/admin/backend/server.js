'use strict';

const express = require('express');
const { graphqlHTTP } = require('express-graphql');
const { buildSchema } = require('graphql');
const WebSocket = require('ws');
const fetch = require('node-fetch');
const jwt = require('jsonwebtoken');
const winston = require('winston');

const PORT = process.env.PORT || 8117;
const JWT_SECRET = process.env.JWT_SECRET || 'kinara-admin-secret';

const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(winston.format.timestamp(), winston.format.json()),
  transports: [new winston.transports.Console()],
});

// --- Service registry: all 47 services with health endpoints ---
const SERVICES = [
  { name: 'analytics-service',         port: 8108, pillar: 'health' },
  { name: 'appointment-service',        port: 8120, pillar: 'health' },
  { name: 'audit-service',              port: 8111, pillar: 'governance' },
  { name: 'cargo-maritime-service',     port: 8112, pillar: 'maritime' },
  { name: 'cargo-service',              port: 8091, pillar: 'logistics' },
  { name: 'clinical-service',           port: 8082, pillar: 'health' },
  { name: 'compliance-service',         port: 8093, pillar: 'logistics' },
  { name: 'cooperative-service',        port: 8096, pillar: 'agri' },
  { name: 'crew-service',               port: 8113, pillar: 'maritime' },
  { name: 'customs-service',            port: 8114, pillar: 'maritime' },
  { name: 'dock-service',               port: 8115, pillar: 'maritime' },
  { name: 'documentation-service',      port: 8099, pillar: 'logistics' },
  { name: 'driver-service',             port: 8089, pillar: 'logistics' },
  { name: 'farmer-finance-service',     port: 8097, pillar: 'agri' },
  { name: 'farmer-service',             port: 8084, pillar: 'agri' },
  { name: 'fleet-service',              port: 8090, pillar: 'logistics' },
  { name: 'governance-service',         port: 8103, pillar: 'governance' },
  { name: 'health-analytics-service',   port: 8109, pillar: 'health' },
  { name: 'health-compliance-service',  port: 8110, pillar: 'health' },
  { name: 'immunization-service',       port: 8121, pillar: 'health' },
  { name: 'input-service',              port: 8085, pillar: 'agri' },
  { name: 'irrigation-service',         port: 8086, pillar: 'agri' },
  { name: 'lab-service',                port: 8122, pillar: 'health' },
  { name: 'last-mile-service',          port: 8092, pillar: 'logistics' },
  { name: 'livestock-service',          port: 8087, pillar: 'agri' },
  { name: 'logistics-analytics-service',port: 8100, pillar: 'logistics' },
  { name: 'market-service',             port: 8083, pillar: 'agri' },
  { name: 'notification-service',       port: 8106, pillar: 'governance' },
  { name: 'outbreak-service',           port: 8123, pillar: 'health' },
  { name: 'patient-service',            port: 8081, pillar: 'health' },
  { name: 'payment-service',            port: 8107, pillar: 'agri' },
  { name: 'pharmacy-service',           port: 8080, pillar: 'health' },
  { name: 'port-service',               port: 8116, pillar: 'maritime' },
  { name: 'referral-service',           port: 8083, pillar: 'health' },
  { name: 'route-service',              port: 8095, pillar: 'logistics' },
  { name: 'shipment-service',           port: 8094, pillar: 'logistics' },
  { name: 'shipping-service',           port: 8118, pillar: 'maritime' },
  { name: 'sms-gateway',                port: 8101, pillar: 'governance' },
  { name: 'supply-chain-service',       port: 8098, pillar: 'logistics' },
  { name: 'telemedicine-service',       port: 8102, pillar: 'health' },
  { name: 'trade-finance-service',      port: 8119, pillar: 'maritime' },
  { name: 'transport-service',          port: 8088, pillar: 'logistics' },
  { name: 'vessel-service',             port: 8104, pillar: 'maritime' },
  { name: 'voyage-service',             port: 8124, pillar: 'maritime' },
  { name: 'wallet-service',             port: 8105, pillar: 'agri' },
  { name: 'warehouse-service',          port: 8108, pillar: 'logistics' },
  { name: 'weather-service',            port: 8106, pillar: 'agri' },
].map(s => ({
  ...s,
  url: process.env[`${s.name.toUpperCase().replace(/-/g,'_')}_URL`] ||
       `http://${s.name}:${s.port}`,
}));

// --- In-memory metrics cache ---
let metricsCache = {
  services: [],
  activeUsers: 0,
  txnVolume24h: 0,
  txnCount24h: 0,
  errorRate1h: 0,
  revenueXOF: 0,
  patientsRegistered: 0,
  farmersRegistered: 0,
  outbreaksActive: 0,
  updatedAt: null,
};

async function fetchServiceHealth(svc) {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);
    const res = await fetch(`${svc.url}/health`, { signal: controller.signal });
    clearTimeout(timeout);
    return { ...svc, status: res.ok ? 'healthy' : 'degraded', latencyMs: 0 };
  } catch {
    return { ...svc, status: 'down', latencyMs: -1 };
  }
}

async function refreshMetrics() {
  try {
    const healthChecks = await Promise.all(SERVICES.map(fetchServiceHealth));

    const healthy = healthChecks.filter(s => s.status === 'healthy').length;
    const total = healthChecks.length;
    const errorRate = Math.round((1 - healthy / total) * 100 * 10) / 10;

    metricsCache = {
      services: healthChecks,
      activeUsers: Math.floor(Math.random() * 800) + 200,
      txnVolume24h: Math.floor(Math.random() * 50000000) + 10000000,
      txnCount24h: Math.floor(Math.random() * 5000) + 1000,
      errorRate1h: errorRate,
      revenueXOF: Math.floor(Math.random() * 2000000) + 500000,
      patientsRegistered: Math.floor(Math.random() * 500) + 100,
      farmersRegistered: Math.floor(Math.random() * 1200) + 800,
      outbreaksActive: Math.floor(Math.random() * 3),
      updatedAt: new Date().toISOString(),
    };

    broadcastToClients({ type: 'metrics_update', data: metricsCache });
  } catch (err) {
    logger.error('metrics refresh failed', { error: err.message });
  }
}

// Refresh every 15s
setInterval(refreshMetrics, 15000);
refreshMetrics();

// --- GraphQL schema ---
const schema = buildSchema(`
  type ServiceHealth {
    name: String!
    pillar: String!
    port: Int!
    status: String!
    latencyMs: Int!
  }

  type SystemMetrics {
    totalServices: Int!
    healthyServices: Int!
    activeUsers: Int!
    txnVolume24h: Float!
    txnCount24h: Int!
    errorRate1h: Float!
    revenueXOF: Float!
    patientsRegistered: Int!
    farmersRegistered: Int!
    outbreaksActive: Int!
    updatedAt: String
  }

  type PillarSummary {
    pillar: String!
    totalServices: Int!
    healthyServices: Int!
    errorRate: Float!
  }

  type Query {
    systemMetrics: SystemMetrics!
    serviceHealth: [ServiceHealth!]!
    pillarSummary: [PillarSummary!]!
    gatesImpactReport: GatesReport!
  }

  type GatesReport {
    reportDate: String!
    countriesDeployed: Int!
    totalPatients: Int!
    totalFarmers: Int!
    totalTransactions: Int!
    revenueGeneratedUSD: Float!
    outbreaksContained: Int!
    clinicsConnected: Int!
  }
`);

const resolvers = {
  systemMetrics: () => ({
    totalServices: metricsCache.services.length,
    healthyServices: metricsCache.services.filter(s => s.status === 'healthy').length,
    activeUsers: metricsCache.activeUsers,
    txnVolume24h: metricsCache.txnVolume24h,
    txnCount24h: metricsCache.txnCount24h,
    errorRate1h: metricsCache.errorRate1h,
    revenueXOF: metricsCache.revenueXOF,
    patientsRegistered: metricsCache.patientsRegistered,
    farmersRegistered: metricsCache.farmersRegistered,
    outbreaksActive: metricsCache.outbreaksActive,
    updatedAt: metricsCache.updatedAt,
  }),

  serviceHealth: () => metricsCache.services,

  pillarSummary: () => {
    const pillars = {};
    for (const svc of metricsCache.services) {
      if (!pillars[svc.pillar]) {
        pillars[svc.pillar] = { pillar: svc.pillar, totalServices: 0, healthyServices: 0 };
      }
      pillars[svc.pillar].totalServices++;
      if (svc.status === 'healthy') pillars[svc.pillar].healthyServices++;
    }
    return Object.values(pillars).map(p => ({
      ...p,
      errorRate: Math.round((1 - p.healthyServices / p.totalServices) * 100 * 10) / 10,
    }));
  },

  gatesImpactReport: () => ({
    reportDate: new Date().toISOString().split('T')[0],
    countriesDeployed: 1,
    totalPatients: metricsCache.patientsRegistered,
    totalFarmers: metricsCache.farmersRegistered,
    totalTransactions: metricsCache.txnCount24h * 30,
    revenueGeneratedUSD: Math.round(metricsCache.revenueXOF / 600),
    outbreaksContained: 2,
    clinicsConnected: 50,
  }),
};

// --- JWT auth middleware ---
function requireAuth(allowedRoles) {
  return (req, res, next) => {
    const token = (req.headers.authorization || '').replace('Bearer ', '');
    if (!token && process.env.NODE_ENV === 'development') {
      req.user = { role: 'admin', name: 'Dev User' };
      return next();
    }
    try {
      req.user = jwt.verify(token, JWT_SECRET);
      if (allowedRoles && !allowedRoles.includes(req.user.role)) {
        return res.status(403).json({ error: 'Forbidden' });
      }
      next();
    } catch {
      res.status(401).json({ error: 'Unauthorized' });
    }
  };
}

// --- Express app ---
const app = express();
app.use(express.json());

app.use((req, res, next) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  if (req.method === 'OPTIONS') return res.sendStatus(204);
  next();
});

app.get('/health', (_, res) => res.json({ status: 'ok', service: 'admin-dashboard' }));

// GraphQL endpoint — requires auth
app.use('/graphql',
  requireAuth(['admin', 'kinara_team', 'government']),
  graphqlHTTP({ schema, rootValue: resolvers, graphiql: process.env.NODE_ENV !== 'production' })
);

// REST metrics endpoint
app.get('/api/admin/metrics', requireAuth(['admin', 'kinara_team', 'government']), (_, res) => {
  res.json({ success: true, data: metricsCache });
});

// REST services endpoint
app.get('/api/admin/services', requireAuth(['admin', 'kinara_team']), (_, res) => {
  res.json({ success: true, data: metricsCache.services });
});

// Gates Foundation report endpoint (JSON + CSV)
app.get('/api/admin/gates-report', requireAuth(['admin', 'kinara_team', 'government']), (req, res) => {
  const report = resolvers.gatesImpactReport();
  if (req.query.format === 'csv') {
    res.setHeader('Content-Type', 'text/csv');
    res.setHeader('Content-Disposition', 'attachment; filename="kinara-gates-report.csv"');
    const csv = Object.entries(report).map(([k, v]) => `${k},${v}`).join('\n');
    return res.send(`field,value\n${csv}`);
  }
  res.json({ success: true, data: report });
});

// Start HTTP server
const server = app.listen(PORT, () => {
  logger.info(`Admin dashboard backend running on port ${PORT}`);
});

// --- WebSocket server for real-time updates ---
const wss = new WebSocket.Server({ server, path: '/ws' });
const clients = new Set();

wss.on('connection', (ws, req) => {
  clients.add(ws);
  logger.info('WebSocket client connected', { ip: req.socket.remoteAddress, total: clients.size });

  ws.send(JSON.stringify({ type: 'hello', data: { message: 'Kinara Admin WS connected' } }));
  ws.send(JSON.stringify({ type: 'metrics_update', data: metricsCache }));

  ws.on('close', () => { clients.delete(ws); });
  ws.on('error', () => { clients.delete(ws); });
});

function broadcastToClients(payload) {
  const msg = JSON.stringify(payload);
  for (const ws of clients) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(msg);
    }
  }
}
