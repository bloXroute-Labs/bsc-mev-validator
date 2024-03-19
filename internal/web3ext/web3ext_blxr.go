package web3ext

const MEVJs = `
web3._extend({
	property: 'mev',
	methods: [
		new web3._extend.Method({
			name: 'proposedBlock',
			call: 'mev_proposedBlock',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'addRelay',
			call: 'mev_addRelay',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'removeRelay',
			call: 'mev_removeRelay',
			params: 1,
		}),
	],

	properties: [
		new web3._extend.Property({
			name: 'blockNumber',
			getter: 'mev_blockNumber',
			outputFormatter: web3._extend.utils.toDecimal
		}),
	],
});
`
