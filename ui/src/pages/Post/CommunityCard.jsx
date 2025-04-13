import PropTypes from 'prop-types';
import React from 'react';
import { useSelector } from 'react-redux';
import CommunityProPic from '../../components/CommunityProPic';
import Link from '../../components/Link';
import MarkdownBody from '../../components/MarkdownBody';
import ShowMoreBox from '../../components/ShowMoreBox';
import JoinButton from '../Community/JoinButton';

const CommunityCard = ({ community }) => {
  const { name } = community;
  const communityURL = `/${name}`;

  // Get the latest community state from Redux
  const latestCommunity = useSelector((state) => 
    state.communities.items[name] || community
  );

  return (
    <div className="card card-sub about-community">
      <div className="about-comm-head">
        <Link to={communityURL} className="about-comm-top">
          <div className="about-comm-profile">
            <CommunityProPic name={latestCommunity.name} proPic={latestCommunity.proPic} size="large" />
          </div>
          <div className="about-comm-head-right">
            <div className="about-comm-name">{latestCommunity.name}</div>
            <div className="about-comm-no-members">
              {`${latestCommunity.noMembers}/11 members`}
            </div>
          </div>
        </Link>
        <div className="about-comm-desc">
          <ShowMoreBox showButton maxHeight="250px">
            <MarkdownBody>{latestCommunity.about}</MarkdownBody>
          </ShowMoreBox>
        </div>
        <div className="about-comm-join">
          <JoinButton community={latestCommunity} />
        </div>
      </div>
    </div>
  );
};

CommunityCard.propTypes = {
  community: PropTypes.object.isRequired,
};

export default CommunityCard;
