package client

// OverrideEndpoints redirects the REST and GraphQL base URLs and returns a
// function that restores the originals. Intended for tests (in this and other
// packages) that point the client at an httptest server.
func OverrideEndpoints(rest, graphql string) func() {
	origRest, origGraphQL := baseURL, graphqlURL
	baseURL, graphqlURL = rest, graphql
	return func() { baseURL, graphqlURL = origRest, origGraphQL }
}
