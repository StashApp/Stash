import {
  CriterionModifier,
} from "../../../core/generated-graphql";
import { Criterion, CriterionType, StringCriterion, NumberCriterion } from "./criterion";
import { FavoriteCriterion } from "./favorite";
import { HasMarkersCriterion } from "./has-markers";
import { IsMissingCriterion } from "./is-missing";
import { NoneCriterion } from "./none";
import { PerformersCriterion } from "./performers";
import { RatingCriterion } from "./rating";
import { ResolutionCriterion } from "./resolution";
import { StudiosCriterion } from "./studios";
import { DvdsCriterion } from "./dvds";
import { TagsCriterion } from "./tags";

export function makeCriteria(type: CriterionType = "none") {
  switch (type) {
    case "none": return new NoneCriterion();
    case "rating": return new RatingCriterion();
    case "resolution": return new ResolutionCriterion();
    case "favorite": return new FavoriteCriterion();
    case "hasMarkers": return new HasMarkersCriterion();
    case "isMissing": return new IsMissingCriterion();
    case "tags": return new TagsCriterion("tags");
    case "sceneTags": return new TagsCriterion("sceneTags");
    case "performers": return new PerformersCriterion();
    case "studios": return new StudiosCriterion();
    case "dvds": return new DvdsCriterion();
    case "birth_year":
    case "age":
        var ret = new NumberCriterion(type, type);
        // null/not null doesn't make sense for these criteria
        ret.modifierOptions = [
          Criterion.getModifierOption(CriterionModifier.Equals),
          Criterion.getModifierOption(CriterionModifier.NotEquals),
          Criterion.getModifierOption(CriterionModifier.GreaterThan),
          Criterion.getModifierOption(CriterionModifier.LessThan)
        ];
        return ret;
    case "ethnicity": 
    case "country":
    case "eye_color":
    case "height":
    case "measurements":
    case "fake_tits":
    case "career_length":
    case "tattoos":
    case "piercings":
    case "aliases":
      return new StringCriterion(type, type);
  }
}
