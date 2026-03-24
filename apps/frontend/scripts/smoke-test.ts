/**
 * Smoke Test Script
 * 
 * Verifies critical API contracts and system health.
 * Run with: npx ts-node scripts/smoke-test.ts
 */

import axios from 'axios';

const API_URL = 'http://localhost:8080/api/v1';

async function runSmokeTests() {
    console.log('🚀 Starting ARC-Hawk Smoke Tests...');
    let passed = 0;
    let failed = 0;

    const test = async (name: string, fn: () => Promise<void>) => {
        try {
            process.stdout.write(`Testing ${name}... `);
            await fn();
            console.log('✅ PASS');
            passed++;
        } catch (e: any) {
            console.log('❌ FAIL');
            console.error(`   Error: ${e.message}`);
            if (e.response) {
                console.error(`   Status: ${e.response.status}`);
                // Inspect data structure to debug
                console.error(`   Data Keys: ${Object.keys(e.response.data)}`);
                if (e.response.data.data) {
                    console.error(`   Data.Data Keys: ${Object.keys(e.response.data.data)}`);
                } else {
                    console.error('   No .data property in response');
                }
            }
            failed++;
        }
    };

    // 1. Health Check
    await test('Backend Health', async () => {
        const res = await axios.get('http://localhost:8080/health');
        if (res.status !== 200) throw new Error(`Status ${res.status}`);
        if (res.data.status !== 'healthy') throw new Error(`Unhealthy status: ${res.data.status}`);
    });

    // 2. Assets Endpoint
    await test('Get Assets', async () => {
        const res = await axios.get(`${API_URL}/assets`);
        if (res.status !== 200) throw new Error(`Status ${res.status}`);

        if (!res.data.data) {
            console.log('Received:', JSON.stringify(res.data, null, 2));
        }
        if (!Array.isArray(res.data.data)) throw new Error('Invalid structure: data array missing');
    });

    // 3. Findings Endpoint
    await test('Get Findings', async () => {
        const res = await axios.get(`${API_URL}/findings`);
        if (res.status !== 200) throw new Error(`Status ${res.status}`);
        if (!res.data.data || !Array.isArray(res.data.data.findings)) throw new Error('Invalid structure');
    });

    console.log('\n--- Summary ---');
    console.log(`Total: ${passed + failed}`);
    console.log(`Passed: ${passed}`);
    console.log(`Failed: ${failed}`);

    if (failed > 0) process.exit(1);
}

runSmokeTests();
