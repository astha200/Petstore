import { gql } from "@apollo/client";

export const AVAILABLE_PETS = gql`
  query AvailablePets($store: String!) {
    availablePets(store: $store) {
      id
      name
      species
      age
      pictureUrl
      description
    }
  }
`;

export const PURCHASE_PET = gql`
  mutation PurchasePet($store: String!, $petId: ID!) {
    purchasePet(store: $store, petId: $petId) {
      id
      name
    }
  }
`;

export const CHECKOUT = gql`
  mutation Checkout($store: String!, $petIds: [ID!]!) {
    checkout(store: $store, petIds: $petIds) {
      purchased {
        id
        name
      }
    }
  }
`;
