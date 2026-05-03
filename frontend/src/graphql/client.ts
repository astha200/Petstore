import { ApolloClient, InMemoryCache, HttpLink, ApolloLink } from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { readCredentialsHeader } from "../auth/AuthProvider";

const GRAPHQL_URL = import.meta.env.VITE_GRAPHQL_URL ?? "http://localhost:8080/query";

export function createApolloClient(): ApolloClient<unknown> {
  const httpLink = new HttpLink({ uri: GRAPHQL_URL });

  const authLink = setContext((_, { headers }) => {
    const auth = readCredentialsHeader();
    return {
      headers: {
        ...headers,
        ...(auth ? { authorization: auth } : {}),
      },
    };
  });

  return new ApolloClient({
    link: ApolloLink.from([authLink, httpLink]),
    cache: new InMemoryCache(),
    defaultOptions: {
      watchQuery: { fetchPolicy: "cache-and-network" },
    },
  });
}
