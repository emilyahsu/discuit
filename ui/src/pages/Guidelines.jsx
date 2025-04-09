import StaticPage from '../components/StaticPage';

const Guidelines = () => {
  return (
    <StaticPage className="page-guidelines" title="Guidelines">
      <main className="document">
        <h1>Seddit site guidelines</h1>
        <p>
          For the purpose of keeping this place civil and friendly, Seddit will have certain
          rules and guidelines that all users must follow. These rules are not meant to
          restrict your freedom of expression, but rather to ensure that everyone can
          participate in discussions without fear of harassment or abuse.
        </p>
        <p>
          Seddit, being a forum of forums, will naturally have two levels of moderation: site-wide
          moderation (by admins) and community-level moderation (by moderators). The site-wide
          rules are the minimum rules that all communities must follow. Communities can have
          their own additional rules, but they cannot override or ignore the site-wide rules.
        </p>
        <h2>Site-wide rules:</h2>
        <ol>
          <li>
            <strong>No spam.</strong> Sharing anything you've created or done on Seddit is
            fine, but don't spam it across multiple
            communities. And don't use Seddit solely for self-promotion.
          </li>
          <li>
            <strong>No porn.</strong> Please don't post anything with sexually explicit material,
            including nudity and depictions of sexual acts.
          </li>
          <li>
            <p>
              <strong>No racism or hate.</strong> Don't post anything that promotes violence or
              discrimination against a group of people based on race, ethnicity, sex, gender,
              religion, nationality, or sexual orientation.
            </p>
            <p>
              Respectfully talking about, for example, racial or national differences, is okay.
              Joking about, say, different cultural idiosyncrasies is also okay, provided that it's
              not coming from a place of hatred.
            </p>
            <p>
              The use of racial slurs, on the other hand, is not okay; nor is promoting the idea
              that a particular race is superior or inferior to all others. Criticism is okay,
              hatred is not.
            </p>
          </li>
          <li>
            <strong>No harassment of other users.</strong> Disagreements are normal and expected,
            but attempts to shutdown discourse, name calling, repetitive comments, threatening
            people, and organizing campaigns of harassment are not allowed.
          </li>
          <li>
            <strong>No doxing.</strong> Don't share someone else's private information (such as
            their phone number or address) without their explicit consent. And don't threaten to do
            so either.
          </li>
          <li>
            <strong>No encouraging harmful behavior.</strong> Don't post anything that encourages
            harmful behavior, like suicide or self-harm.
          </li>
          <li>
            <strong>No brigading.</strong> Don't organize campaigns to 1) downvote posts and
            comments of a community you don't like or 2) to leave hostile posts and comments in a
            community you don't like. If you don't like the rules of a particular community or how
            it moderates content, you can create a new community as <i>you</i> like.
          </li>
        </ol>
        <p>
          Breaking these rules may result in the offending material being removed and/or temporary
          or permanent suspension of the user accounts involved.
        </p>
        <p>
          Seddit is a work in progress, and these rules are not set in stone. They may change in
          the future based on community feedback and our own observations.
        </p>
      </main>
    </StaticPage>
  );
};

export default Guidelines;
