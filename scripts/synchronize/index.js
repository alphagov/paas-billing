const axios = require('axios');
const knex = require('knex')({ client: 'pg' });

const API = process.argv[2];
const TOKEN = process.argv[3];

const cf = axios.create({
	baseURL: API,
	timeout: 5000,
	headers: {
		'Authorization': TOKEN,
		'Content-Type': 'application/json',
	},
  });

async function run() {
	const orgs = await cf.get('/v3/organizations?per_page=5000');

	return orgs.data.resources.map(org => {
		const { guid, name } = org;
		const owner = org.metadata.annotations['owner'];

		return knex('orgs').where({ guid }).update({ name, owner });
	}).join('\n');
}

/**
 * Script requires to be `cf login` into the platform.
 * 
 * Run:
 * node index.js "$(cf api | grep endpoint | awk '{print $3}')" "$(cf oauth-token)"
 */
run()
	.then(console.log)
	.catch(console.error)
