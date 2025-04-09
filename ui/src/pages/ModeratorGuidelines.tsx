import { Link } from 'react-router-dom';
import StaticPage from '../components/StaticPage';

const ModeratorGuidelines = () => {
  return (
    <StaticPage className="page-guidelines" title="Guidelines">
      <main className="document">
        <h1>Seddit moderator guidelines</h1>
        <p>
          Moderators are entrusted with a lot of power and responsibility within the Seddit
          community. These guidelines are designed to help moderators understand their role
          and responsibilities, and to ensure that they act in the best interests of their
          communities.
        </p>
        <h2>General Guidelines</h2>
        <p>
          Seddit moderators are expected to follow and uphold Seddit's{' '}
          <Link to="/guidelines">site guidelines.</Link> Any user content in violation of Seddit's
          site guidelines should be removed immediately. In cases of severe or repeated
          violation of Seddit's site guidelines, moderators are responsible for escalating these
          issues to the admin team.
        </p>
        <h2>Moderation Responsibilities</h2>
        <ul>
          <li>Remove content that violates site or community guidelines</li>
          <li>Respond to user reports in a timely manner</li>
          <li>Maintain a welcoming and inclusive community environment</li>
          <li>Communicate clearly with users about moderation decisions</li>
          <li>Ban users based on participation in other Seddit communities.</li>
        </ul>
        <h2>Moderation Best Practices</h2>
        <ul>
          <li>Be consistent in applying rules and guidelines</li>
          <li>Document significant moderation decisions</li>
          <li>Communicate with users respectfully and professionally</li>
          <li>Work collaboratively with other moderators</li>
          <li>Seek admin guidance when dealing with complex situations</li>
        </ul>
        <h2>Appeals Process</h2>
        <p>
          When a user is reprimanded for violating Seddit community or site rules—either by removing
          content or being banned—they have the right to appeal the decision. Moderators should
          be prepared to explain their decisions and work with users to resolve conflicts
          whenever possible.
        </p>
        <h2>Admin Oversight</h2>
        <p>
          The Seddit admin team has the authority to remove any moderator for abuse of power, or,
          in rare cases, to override moderation decisions that violate site guidelines or
          principles. This oversight ensures that moderation remains fair and consistent
          across all communities.
        </p>
        <h2>Conclusion</h2>
        <p>
          Please remember to be kind. Though we only see a username and an avatar on Seddit, always
          remember that there's a real person behind every interaction. Your role as a moderator
          is to help create and maintain a positive community environment where everyone feels
          welcome and respected.
        </p>
      </main>
    </StaticPage>
  );
};

export default ModeratorGuidelines;
