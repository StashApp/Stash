// NOTE: add new enum values to the end, to ensure existing data

// is not impacted
export enum DisplayMode {
  Grid,
  List,
  Wall,
  Tagger,
}

export interface ILabeledId {
  id: string;
  label: string;
}

export interface ILabeledValue {
  label: string;
  value: string;
}

export interface IHierarchicalLabelValue {
  items: ILabeledId[];
  depth: number;
}

export interface INumberValue {
  exact: number;
  lower: number;
  upper: number;
}

export function criterionIsHierarchicalLabelValue(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any
): value is IHierarchicalLabelValue {
  return typeof value === "object" && "items" in value && "depth" in value;
}

export function criterionIsNumberValue(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any
): value is INumberValue {
  return (
    typeof value === "object" &&
    "exact" in value &&
    "lower" in value &&
    "upper" in value
  );
}

export function encodeILabeledId(o: ILabeledId) {
  // escape " and \ and by encoding to JSON so that it encodes to JSON correctly down the line
  const adjustedLabel = JSON.stringify(o.label).slice(1, -1);
  return { ...o, label: encodeURIComponent(adjustedLabel) };
}

export interface IOptionType {
  id: string;
  name?: string;
  image_path?: string;
}

export type CriterionType =
  | "none"
  | "path"
  | "rating"
  | "organized"
  | "o_counter"
  | "resolution"
  | "average_resolution"
  | "duration"
  | "favorite"
  | "hasMarkers"
  | "sceneIsMissing"
  | "imageIsMissing"
  | "performerIsMissing"
  | "galleryIsMissing"
  | "tagIsMissing"
  | "studioIsMissing"
  | "movieIsMissing"
  | "tags"
  | "sceneTags"
  | "performerTags"
  | "tag_count"
  | "performers"
  | "studios"
  | "movies"
  | "galleries"
  | "birth_year"
  | "age"
  | "ethnicity"
  | "country"
  | "hair_color"
  | "eye_color"
  | "height"
  | "weight"
  | "measurements"
  | "fake_tits"
  | "career_length"
  | "tattoos"
  | "piercings"
  | "aliases"
  | "gender"
  | "parent_studios"
  | "scene_count"
  | "marker_count"
  | "image_count"
  | "gallery_count"
  | "performer_count"
  | "death_year"
  | "url"
  | "stash_id"
  | "interactive"
  | "name"
  | "details"
  | "title"
  | "oshash"
  | "checksum"
  | "sceneChecksum"
  | "galleryChecksum"
  | "phash"
  | "director"
  | "synopsis";
