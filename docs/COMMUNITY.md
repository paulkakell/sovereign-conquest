# Community guide

## Choose a channel

| Need | Channel | Example |
| --- | --- | --- |
| Reproducible defect | Bug report issue form | A trade command returns the wrong cargo total |
| Concrete implementation proposal | Feature request issue form | A route preview with observable acceptance criteria |
| Missing or incorrect instructions | Documentation issue form | An installation step omits a required setting |
| Gameplay or hosting help | Q&A discussion | How to pin the server image for an upgrade |
| Early concept or balance feedback | Ideas discussion | Improving a returning player's first session |
| Project news | Announcements discussion | A release or scheduled maintenance notice |
| Conversation and feedback | General discussion | Introductions or a play-session retrospective |
| Community work | Show and tell discussion | A trading strategy, public tool, or server project |
| Community preference survey | Polls discussion | Which map presentation is easiest to read |
| Vulnerability or exploitable cheating | Private security report | A way to access another player's assets |

Start at [Discussions](https://github.com/paulkakell/sovereign-conquest/discussions)
or the [issue chooser](https://github.com/paulkakell/sovereign-conquest/issues/new/choose).
Read [SECURITY.md](../SECURITY.md) before reporting a suspected vulnerability.

Be specific, respectful, and willing to explain the player problem. Critique
ideas without attacking people. Keep credentials, personal data, and confidential
corporation intelligence out of public posts. Search for related work and link it.
A popular discussion is feedback, not a promise of implementation or a release date.

## Issues and roadmap

The bug form requests version, area, reproduction, expected and actual behavior,
and environment. The feature form requests a problem, proposed behavior, and
acceptance criteria. The documentation form requests the page, version, and gap.
Blank issues are disabled in the chooser so contributors can find the right form;
the security policy documents a minimal General-discussion fallback if private
reporting is unavailable.

Link an existing [roadmap](ROADMAP.md) packet such as `SC-I04` when relevant. Keep
planning status accurate: a submitted issue or discussion does not complete a packet.
Maintainers can apply issue labels during triage. Forms do not depend on labels
being present in a repository copy.

## Discussions configuration

Discussions is enabled. The existing categories are Announcements, General,
Ideas, Polls, Q&A, and Show and tell. Their slugs were checked against the live
repository on 2026-09-27. Forms are provided for every category except Polls,
which uses GitHub's native poll composer.

The filenames in `.github/DISCUSSION_TEMPLATE/` match those category slugs.
Renaming a category may change its slug; update the filename and incoming links
at the same time. Q&A supports accepted answers. Announcements is intended for
maintainer release and project news. Existing category permissions are preserved.

To verify configuration after merging:

1. Open the issue chooser and preview each of the three forms.
2. Open a new discussion in Q&A, Ideas, and the other non-poll categories; check
   the fields without submitting test posts.
3. Open Security and confirm that the policy and Report a vulnerability are shown.
4. Check the Sponsor destination as described below.

These files take effect on the default branch. Private reporting and Discussions
are repository settings; copying files to another repository does not enable them.

## Sponsorship

`.github/FUNDING.yml` names the repository owner, `paulkakell`, as the GitHub
Sponsors recipient. No external donation account or payment URL is assumed.
GitHub requires that account to have an active Sponsors profile before it can
receive sponsorships. The funding file does not enroll the account, configure
payouts, create sponsorship tiers, or establish any sponsor benefits.

The repository setup includes the funding destination; account enrollment and
payment acceptance must be checked separately at
[GitHub Sponsors](https://github.com/sponsors/paulkakell). If the destination is
unavailable, the owner must finish Sponsors setup or supply a verified alternative
funding URL before the button can serve as a working payment route.

To use an approved external destination, add a `custom` entry with its HTTPS URL
to `FUNDING.yml`. Verify ownership and the final destination before publishing.
Do not insert a guessed payment handle. Funding does not change the roadmap status.

## Maintaining this setup

Keep public bug and help forms separate from private security reporting. Review
templates when deployment instructions or the supported release policy change.
Do not ask reporters for secrets or complete production data exports. Use
versioned release notes and the changelog to communicate delivered changes.

GitHub references:

- [Issue forms and chooser configuration](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [Discussion category form syntax](https://docs.github.com/en/discussions/managing-discussions-for-your-community/syntax-for-discussion-category-forms)
- [Sponsor button configuration](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository)
