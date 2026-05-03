export type Species = "CAT" | "DOG" | "FROG";

export interface Pet {
  id: string;
  name: string;
  species: Species;
  age: number;
  pictureUrl: string;
  description: string;
}
