package web3ext

const MEVJs = `
web3._extend({
	property: 'mev',
	methods: [
		new web3._extend.Method({
			name: 'registerValidator',
			call: 'mev_registerValidator',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'proposedBlock',
			call: 'mev_proposedBlock',
			params: 1,
		}),
	]
});
`
