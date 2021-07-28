## GraphQL handler for gin
![go workflow](https://github.com/asif-mahmud/graphqlgin/actions/workflows/go.yml/badge.svg)

This is a small package to provide a [GraphQL](https://graphql.org/) handler that can be used with
[Gin Framework](https://github.com/gin-gonic/gin).

### Features
1. Fully tested.
2. Supports context managers so user can add their application specific data to be used in resolver
   functions.
3. Supports file upload out of the box.
4. Fully compliant with [GraphQL multipart specification](https://github.com/jaydenseric/graphql-multipart-request-spec),
   so client libraries like [Apollo Upload Client](https://www.npmjs.com/package/apollo-upload-client) will work
   out of the box.
5. Allows adding additional http headers either by gin middleware, or right from the resolver functions.

### Installation
To add the package to your project run -

```
go get -u github.com/asif-mahmud/graphqlgin
```

### Documentation

godoc: [https://pkg.go.dev/github.com/asif-mahmud/graphqlgin](https://pkg.go.dev/github.com/asif-mahmud/graphqlgin)
examples: [https://pkg.go.dev/github.com/asif-mahmud/graphqlgin#pkg-examples](https://pkg.go.dev/github.com/asif-mahmud/graphqlgin#pkg-examples)
