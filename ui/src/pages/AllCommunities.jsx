import clsx from 'clsx';
import React, { useEffect, useRef, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import Button from '../components/Button';
import CommunityProPic from '../components/CommunityProPic';
import Dropdown from '../components/Dropdown';
import Feed from '../components/Feed';
import Input from '../components/Input';
import MarkdownBody from '../components/MarkdownBody';
import MiniFooter from '../components/MiniFooter';
import ShowMoreBox from '../components/ShowMoreBox';
import Sidebar from '../components/Sidebar';
import { mfetchjson } from '../helper';
import { FeedItem } from '../slices/feedsSlice';
import {
  allCommunitiesSearchQueryChanged,
  allCommunitiesSortChanged,
  snackAlertError,
  createCommunityModalOpened,
} from '../slices/mainSlice';
import { SVGClose, SVGSearch } from '../SVGs';
import LoginForm from '../views/LoginForm';
import JoinButton from './Community/JoinButton';
import { isInfiniteScrollingDisabled } from './Settings/devicePrefs';
import CreateCommunity from '../components/CreateCommunity';
import PropTypes from 'prop-types';

const prepareText = (isMobile = false) => {
  const x = isMobile ? 'by filling out the form below' : 'by clicking on the button below';
  return `Communities are currently available only on a per request
    basis. You can request one ${x}, and if you seem
    reasonable and trustworthy, the requested community will be created and you will
    be added as a moderator of that community.`;
};

const AllCommunities = () => {
  const dispatch = useDispatch();

  const user = useSelector((state) => state.main.user);
  const loggedIn = user !== null;

  const searchQuery = useSelector((state) => state.main.allCommunitiesSearchQuery);
  const [isSearching, setIsSearching] = useState(searchQuery !== '');
  const setSearchQuery = (query) => {
    dispatch(allCommunitiesSearchQueryChanged(query));
  };
  useEffect(() => {
    if (!isSearching) {
      setSearchQuery('');
    }
  }, [isSearching]);

  const sort = useSelector((state) => state.main.allCommunitiesSort);
  const setSort = (sort) => {
    dispatch(allCommunitiesSortChanged(sort));
  };

  const fetchCommunities = async (next) => {
    const res = await mfetchjson(`/api/communities?sort=${sort}`);
    const items = res.map((community) => new FeedItem(community, 'community', community.id));
    return {
      items: items,
      next: null,
    };
  };

  const handleRenderItem = (item, index) => {
    if (
      searchQuery !== '' &&
      !item.item.name.toLowerCase().includes(searchQuery.trim().toLowerCase())
    ) {
      return null;
    }
    return <ListItem community={item.item} />;
  };

  const renderSearchBox = () => {
    return (
      <div className="communities-search">
        <Input
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
          autoFocus
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              setIsSearching(false);
            }
          }}
        />
      </div>
    );
  };

  const renderSortDropdown = () => {
    const sortOptions = {
      new: 'Latest',
      size: 'Popular',
      name_asc: 'A-Z',
      name_dsc: 'Z-A',
    };
    return (
      <Dropdown target={<Button>{sortOptions[sort]}</Button>} aligned="right">
        <div className="dropdown-list">
          {Object.keys(sortOptions)
            .filter((key) => key !== sort)
            .map((key) => (
              <Button className="button-clear dropdown-item" onClick={() => setSort(key)} key={key}>
                {sortOptions[key]}
              </Button>
            ))}
        </div>
      </Dropdown>
    );
  };

  return (
    <div className="page-content page-comms wrap page-grid">
      <Sidebar />
      <main>
        <div className="page-comms-header card card-padding">
          <div className="left">{isSearching ? renderSearchBox() : <h1>All communities</h1>}</div>
          <div className="right">
            <Button
              className={clsx('comms-search-button', !isSearching && 'is-search-svg')}
              icon={isSearching ? <SVGClose /> : <SVGSearch />}
              onClick={() => setIsSearching((v) => !v)}
            />
            {!isSearching && renderSortDropdown()}
            {!isSearching && (
              <button 
                className="button-main is-m comms-new-button" 
                onClick={() => dispatch(createCommunityModalOpened())}
              >
                New
              </button>
            )}
          </div>
        </div>
        <div className="comms-list">
          <Feed
            feedId={'all-communities-' + sort}
            onFetch={fetchCommunities}
            onRenderItem={handleRenderItem}
            infiniteScrollingDisabled={isInfiniteScrollingDisabled()}
            noMoreItemsText="Nothing to show"
          />
        </div>
      </main>
      <aside className="sidebar-right">
        {!loggedIn && (
          <div className="card card-sub card-padding">
            <LoginForm />
          </div>
        )}
        <CommunityCreationCard />
        <MiniFooter />
      </aside>
    </div>
  );
};

AllCommunities.propTypes = {};

const CommunityCreationCard = () => {
  const [open, setOpen] = useState(false);
  const handleClose = () => setOpen(false);

  return (
    <div className="card card-sub card-padding home-welcome">
      <div className="home-welcome-join">New communities</div>
      <div className="home-welcome-subtext">
        Create a new community to discuss topics you're passionate about.
      </div>
      <div className="home-welcome-buttons">
        <button className="button-main" onClick={() => setOpen(true)}>
          Create a community
        </button>
      </div>
      <CreateCommunity open={open} onClose={handleClose} />
    </div>
  );
};

const ListItem = React.memo(function ListItem({ community }) {
  const to = `/${community.name}`;
  const history = useHistory();
  const ref = useRef();

  // Get the latest community state from Redux using the community name as the key
  const latestCommunity = useSelector((state) => 
    state.communities.items[community.name] || community
  );

  const handleClick = (e) => {
    if (e.target.tagName !== 'BUTTON') {
      history.push(to);
    }
  };

  return (
    <div
      ref={ref}
      className="comms-list-item card"
      onClick={handleClick}
      style={{ minHeight: '100px' }}
    >
      <div className="comms-list-item-left">
        <CommunityProPic
          className="is-no-hover"
          name={latestCommunity.name}
          proPic={latestCommunity.proPic}
          size="large"
        />
      </div>
      <div className="comms-list-item-right">
        <div className="comms-list-item-name">
          <a
            href={to}
            className="link-reset comms-list-item-name-name"
            onClick={(e) => e.preventDefault()}
          >
            {latestCommunity.name}
          </a>
          <JoinButton className="comms-list-item-join" community={latestCommunity} />
        </div>
        <div className="comms-list-item-count">{`${latestCommunity.noMembers}/11 members`}</div>
        <div className="comms-list-item-about">
          <ShowMoreBox maxHeight="120px">
            <MarkdownBody>{latestCommunity.about}</MarkdownBody>
          </ShowMoreBox>
        </div>
      </div>
    </div>
  );
});

ListItem.propTypes = {
  community: PropTypes.object.isRequired,
};

export default AllCommunities;
