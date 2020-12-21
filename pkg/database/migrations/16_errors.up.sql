CREATE TABLE `scene_errors` (
  `scene_id` integer,
  `related_scene_id` integer,
  `error_type` varchar(20) not null,
  `recurring` varchar(2),
  `details` varchar(255) not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
  foreign key(`related_scene_id`) references `scenes`(`id`) on delete CASCADE
);

