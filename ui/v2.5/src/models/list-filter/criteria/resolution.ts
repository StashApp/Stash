import {
  ResolutionCriterionInput,
  ResolutionEnum,
  CriterionModifier,
} from "src/core/generated-graphql";
import { CriterionType } from "../types";
import { CriterionOption, StringCriterion } from "./criterion";

abstract class AbstractResolutionCriterion extends StringCriterion {
  protected toCriterionInput(): ResolutionCriterionInput | undefined {
    switch (this.value) {
      case "144p":
        return {
          value: ResolutionEnum.VeryLow,
          modifier: this.modifier,
        };
      case "240p":
        return {
          value: ResolutionEnum.Low,
          modifier: this.modifier,
        };
      case "360p":
        return {
          value: ResolutionEnum.R360P,
          modifier: this.modifier,
        };
      case "480p":
        return {
          value: ResolutionEnum.Standard,
          modifier: this.modifier,
        };
      case "540p":
        return {
          value: ResolutionEnum.WebHd,
          modifier: this.modifier,
        };
      case "720p":
        return {
          value: ResolutionEnum.StandardHd,
          modifier: this.modifier,
        };
      case "1080p":
        return {
          value: ResolutionEnum.FullHd,
          modifier: this.modifier,
        };
      case "1440p":
        return {
          value: ResolutionEnum.QuadHd,
          modifier: this.modifier,
        };
      case "1920p":
        return {
          value: ResolutionEnum.VrHd,
          modifier: this.modifier,
        };
      case "4k":
        return {
          value: ResolutionEnum.FourK,
          modifier: this.modifier,
        };
      case "5k":
        return {
          value: ResolutionEnum.FiveK,
          modifier: this.modifier,
        };
      case "6k":
        return {
          value: ResolutionEnum.SixK,
          modifier: this.modifier,
        };
      case "8k":
        return {
          value: ResolutionEnum.EightK,
          modifier: this.modifier,
        };
      // no default
    }
  }
}

class ResolutionCriterionOptionType extends CriterionOption {
  constructor(value: CriterionType) {
    super({
      messageID: value,
      type: value,
      parameterName: value,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.GreaterThan,
        CriterionModifier.LessThan,
      ],
      options: [
        "144p",
        "240p",
        "360p",
        "480p",
        "540p",
        "720p",
        "1080p",
        "1440p",
        "4k",
        "5k",
        "6k",
        "8k",
      ],
    });
  }
}

export const ResolutionCriterionOption = new ResolutionCriterionOptionType(
  "resolution"
);
export class ResolutionCriterion extends AbstractResolutionCriterion {
  constructor() {
    super(ResolutionCriterionOption);
  }
}

export const AverageResolutionCriterionOption = new ResolutionCriterionOptionType(
  "average_resolution"
);

export class AverageResolutionCriterion extends AbstractResolutionCriterion {
  constructor() {
    super(AverageResolutionCriterionOption);
  }
}
